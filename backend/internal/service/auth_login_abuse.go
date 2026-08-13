package service

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/yuchen0x1/sub3api-relay/internal/pkg/errors"
)

const (
	defaultLoginFailureWindow      = 15 * time.Minute
	defaultLoginLockDuration       = 30 * time.Minute
	defaultLoginIPFailureThreshold = 5
	defaultLoginIDFailureThreshold = 10
	defaultLoginAbuseMaxEntries    = 20000
)

type loginAbuseBucket struct {
	failures     int
	firstFailure time.Time
	lastSeen     time.Time
	lockedUntil  time.Time
}

// LoginLockout describes a temporary server-side login block.
type LoginLockout struct {
	Scope      string
	RetryAfter time.Duration
}

// LoginAbuseProtector blocks repeated failed password attempts by client IP and
// by login identifier. It is intentionally in-process so deployments still have
// protection even when Redis is unavailable.
type LoginAbuseProtector struct {
	mu                 sync.Mutex
	now                func() time.Time
	ipFailures         map[string]*loginAbuseBucket
	identifierFailures map[string]*loginAbuseBucket
	window             time.Duration
	lockDuration       time.Duration
	ipFailureThreshold int
	idFailureThreshold int
	maxEntries         int
}

func NewLoginAbuseProtector() *LoginAbuseProtector {
	return &LoginAbuseProtector{
		now:                time.Now,
		ipFailures:         make(map[string]*loginAbuseBucket),
		identifierFailures: make(map[string]*loginAbuseBucket),
		window:             defaultLoginFailureWindow,
		lockDuration:       defaultLoginLockDuration,
		ipFailureThreshold: defaultLoginIPFailureThreshold,
		idFailureThreshold: defaultLoginIDFailureThreshold,
		maxEntries:         defaultLoginAbuseMaxEntries,
	}
}

// CheckLoginLocked returns a lockout when this IP or identifier is already blocked.
func (p *LoginAbuseProtector) CheckLoginLocked(clientIP, identifier string) (LoginLockout, bool) {
	if p == nil {
		return LoginLockout{}, false
	}
	now := p.now()
	p.mu.Lock()
	defer p.mu.Unlock()

	if lock, ok := p.lockoutForBucket(now, p.ipFailures[loginIPKey(clientIP)], "ip"); ok {
		return lock, true
	}
	if key := loginIdentifierKey(identifier); key != "" {
		if lock, ok := p.lockoutForBucket(now, p.identifierFailures[key], "identifier"); ok {
			return lock, true
		}
	}
	return LoginLockout{}, false
}

// RecordLoginFailure records one failed password attempt and returns a lockout
// when the attempt crosses either threshold.
func (p *LoginAbuseProtector) RecordLoginFailure(clientIP, identifier string) (LoginLockout, bool) {
	if p == nil {
		return LoginLockout{}, false
	}
	now := p.now()
	p.mu.Lock()
	defer p.mu.Unlock()

	var strongest LoginLockout
	locked := false
	if lock, ok := p.recordFailure(now, p.ipFailures, loginIPKey(clientIP), p.ipFailureThreshold, "ip"); ok {
		strongest = lock
		locked = true
	}
	if key := loginIdentifierKey(identifier); key != "" {
		if lock, ok := p.recordFailure(now, p.identifierFailures, key, p.idFailureThreshold, "identifier"); ok {
			if !locked || lock.RetryAfter > strongest.RetryAfter {
				strongest = lock
			}
			locked = true
		}
	}
	p.pruneIfNeeded(now)
	return strongest, locked
}

// ResetLoginFailures clears failed-attempt counters after a successful login.
func (p *LoginAbuseProtector) ResetLoginFailures(clientIP, identifier string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.ipFailures, loginIPKey(clientIP))
	if key := loginIdentifierKey(identifier); key != "" {
		delete(p.identifierFailures, key)
	}
}

func (p *LoginAbuseProtector) recordFailure(now time.Time, buckets map[string]*loginAbuseBucket, key string, threshold int, scope string) (LoginLockout, bool) {
	if key == "" || threshold <= 0 {
		return LoginLockout{}, false
	}
	bucket := buckets[key]
	if bucket == nil {
		bucket = &loginAbuseBucket{firstFailure: now}
		buckets[key] = bucket
	}
	bucket.lastSeen = now
	if lock, ok := p.lockoutForBucket(now, bucket, scope); ok {
		return lock, true
	}
	if bucket.firstFailure.IsZero() || now.Sub(bucket.firstFailure) > p.window {
		bucket.failures = 0
		bucket.firstFailure = now
	}
	bucket.failures++
	if bucket.failures >= threshold {
		bucket.lockedUntil = now.Add(p.lockDuration)
		return LoginLockout{Scope: scope, RetryAfter: p.lockDuration}, true
	}
	return LoginLockout{}, false
}

func (p *LoginAbuseProtector) lockoutForBucket(now time.Time, bucket *loginAbuseBucket, scope string) (LoginLockout, bool) {
	if bucket == nil || !bucket.lockedUntil.After(now) {
		return LoginLockout{}, false
	}
	return LoginLockout{Scope: scope, RetryAfter: bucket.lockedUntil.Sub(now)}, true
}

func (p *LoginAbuseProtector) pruneIfNeeded(now time.Time) {
	if len(p.ipFailures)+len(p.identifierFailures) <= p.maxEntries {
		return
	}
	p.pruneBuckets(now, p.ipFailures)
	p.pruneBuckets(now, p.identifierFailures)
}

func (p *LoginAbuseProtector) pruneBuckets(now time.Time, buckets map[string]*loginAbuseBucket) {
	staleAfter := p.window + p.lockDuration
	for key, bucket := range buckets {
		if bucket == nil || (!bucket.lockedUntil.After(now) && now.Sub(bucket.lastSeen) > staleAfter) {
			delete(buckets, key)
		}
	}
}

func loginIPKey(clientIP string) string {
	clientIP = strings.TrimSpace(clientIP)
	if clientIP == "" {
		return "unknown"
	}
	return clientIP
}

func loginIdentifierKey(identifier string) string {
	identifier = strings.ToLower(strings.TrimSpace(identifier))
	if identifier == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(identifier))
	return hex.EncodeToString(sum[:])
}

// LoginTemporarilyLockedError builds the public error returned to attackers.
func LoginTemporarilyLockedError(lock LoginLockout) *infraerrors.ApplicationError {
	seconds := retryAfterSeconds(lock.RetryAfter)
	return infraerrors.New(http.StatusTooManyRequests, "AUTH_LOGIN_TEMPORARILY_LOCKED", "too many failed login attempts; try again later").
		WithMetadata(map[string]string{
			"retry_after_seconds": strconv.FormatInt(seconds, 10),
			"scope":               lock.Scope,
		})
}

func retryAfterSeconds(d time.Duration) int64 {
	if d <= 0 {
		d = defaultLoginLockDuration
	}
	seconds := int64(d / time.Second)
	if d%time.Second > 0 {
		seconds++
	}
	if seconds < 1 {
		return 1
	}
	return seconds
}
