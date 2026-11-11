package setup

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/mail"
	"net/netip"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/yuchen0x1/sub3api-relay/internal/pkg/response"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/sysutil"

	"github.com/gin-gonic/gin"
)

// installMutex prevents concurrent installation attempts (TOCTOU protection)
var installMutex sync.Mutex

const setupTokenCookieName = "sub3api_setup_token"

// RegisterRoutes registers setup wizard routes
func RegisterRoutes(r *gin.Engine) {
	setup := r.Group("/setup")
	{
		// Status endpoint is always accessible (read-only)
		setup.GET("/status", getStatus)

		// All modification endpoints are protected by setupGuard
		protected := setup.Group("")
		protected.Use(setupGuard())
		{
			protected.POST("/test-db", testDatabase)
			protected.POST("/test-redis", testRedis)
			protected.POST("/install", install)
			protected.POST("/admin", completeAdminSetup)
		}
	}
}

// SetupStatus represents the current setup state
type SetupStatus struct {
	NeedsSetup    bool   `json:"needs_setup"`
	Step          string `json:"step"`
	OwnerSetup    bool   `json:"owner_setup"`
	TokenRequired bool   `json:"token_required"`
	Mode          string `json:"mode"`
}

// getStatus returns the current setup status
func getStatus(c *gin.Context) {
	ownerSetup := AdminSetupPending()
	mode := "wizard"
	step := "welcome"
	if ownerSetup {
		mode = "owner"
		step = "admin"
	}
	response.Success(c, SetupStatus{
		NeedsSetup:    NeedsSetup(),
		Step:          step,
		OwnerSetup:    ownerSetup,
		TokenRequired: !isLocalSetupRequest(c.Request) && remoteSetupEnabled(),
		Mode:          mode,
	})
}

// setupGuard middleware ensures setup endpoints are only accessible during setup mode
func setupGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !NeedsSetup() {
			response.Error(c, http.StatusForbidden, "Setup is not allowed: system is already installed")
			c.Abort()
			return
		}
		if !remoteSetupEnabled() && !isLocalSetupRequest(c.Request) {
			response.Error(c, http.StatusForbidden, "Remote setup is disabled; use localhost or explicitly enable protected remote setup")
			c.Abort()
			return
		}
		providedToken := strings.TrimSpace(c.GetHeader("X-Setup-Token"))
		if providedToken == "" {
			if cookie, err := c.Request.Cookie(setupTokenCookieName); err == nil {
				providedToken = strings.TrimSpace(cookie.Value)
			}
		}
		if !isLocalSetupRequest(c.Request) && !validRemoteSetupToken(providedToken) {
			response.Error(c, http.StatusForbidden, "Remote setup requires a valid setup token")
			c.Abort()
			return
		}
		if AdminSetupPending() && c.Request.URL.Path != "/setup/admin" {
			response.Error(c, http.StatusConflict, "Infrastructure is ready; complete site owner setup first")
			c.Abort()
			return
		}
		c.Next()
	}
}

func validRemoteSetupToken(provided string) bool {
	expected := strings.TrimSpace(os.Getenv("SETUP_REMOTE_TOKEN"))
	provided = strings.TrimSpace(provided)
	if len(expected) < 32 || len(provided) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

func remoteSetupEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SETUP_ALLOW_REMOTE"))) {
	case "true", "1", "yes":
		return true
	default:
		return false
	}
}

func privateSetupTargetsEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SETUP_ALLOW_PRIVATE_TARGETS"))) {
	case "true", "1", "yes":
		return true
	default:
		return false
	}
}

func isLocalSetupRequest(req *http.Request) bool {
	if req == nil || hasForwardingHeaders(req.Header) || !isLocalSetupHost(req.Host) {
		return false
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(req.RemoteAddr))
	if err != nil {
		host = strings.Trim(strings.TrimSpace(req.RemoteAddr), "[]")
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	addr = addr.Unmap()
	if addr.IsLoopback() {
		return true
	}

	// Docker Desktop/NAT rewrites a host-loopback connection to the
	// container's bridge gateway (for example, 172.18.0.1).  Compose passes
	// the host bind address explicitly so this compatibility path is only
	// enabled when the published port is actually bound to a loopback host
	// address.  It never trusts forwarded headers and it remains disabled for
	// publicly bound deployments.
	return addr.IsPrivate() && isDockerLoopbackBinding()
}

func isDockerLoopbackBinding() bool {
	return isLocalSetupHost(strings.TrimSpace(os.Getenv("SETUP_DOCKER_BIND_HOST")))
}

func isLocalSetupHost(rawHost string) bool {
	host := strings.TrimSpace(rawHost)
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	address, err := netip.ParseAddr(host)
	return err == nil && address.Unmap().IsLoopback()
}

func hasForwardingHeaders(header http.Header) bool {
	for _, name := range []string{
		"Forwarded",
		"X-Forwarded-For",
		"X-Real-IP",
		"CF-Connecting-IP",
		"True-Client-IP",
	} {
		if strings.TrimSpace(header.Get(name)) != "" {
			return true
		}
	}
	return false
}

func resolveSetupTarget(host string, allowPrivate bool) ([]netip.Addr, error) {
	host = strings.TrimSpace(host)
	if !validateHostname(host) {
		return nil, fmt.Errorf("invalid hostname format")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("resolve hostname: %w", err)
	}
	if len(addresses) == 0 {
		return nil, fmt.Errorf("hostname resolved to no addresses")
	}

	approved := make([]netip.Addr, 0, len(addresses))
	for _, address := range addresses {
		address = address.Unmap()
		if err := validateSetupTargetAddress(address, allowPrivate); err != nil {
			return nil, err
		}
		approved = append(approved, address)
	}
	return approved, nil
}

func validateSetupTargetAddress(address netip.Addr, allowPrivate bool) error {
	if !address.IsValid() || address.IsUnspecified() || address.IsMulticast() ||
		address.IsLinkLocalUnicast() || address.IsLinkLocalMulticast() {
		return fmt.Errorf("target address %s is not allowed during web setup", address)
	}
	if address.IsLoopback() || address.IsPrivate() || address.Is4In6() || isSharedAddressSpace(address) {
		if !allowPrivate {
			return fmt.Errorf("private target address %s is not allowed for remote web setup", address)
		}
	}
	return nil
}

func isSharedAddressSpace(address netip.Addr) bool {
	prefix := netip.MustParsePrefix("100.64.0.0/10")
	return prefix.Contains(address)
}

// validateHostname checks if a hostname/IP is safe (no injection characters)
func validateHostname(host string) bool {
	// Allow only alphanumeric, dots, hyphens, and colons (for IPv6)
	validHost := regexp.MustCompile(`^[a-zA-Z0-9.\-:]+$`)
	return validHost.MatchString(host) && len(host) <= 253
}

// validateDBName checks if database name is safe
func validateDBName(name string) bool {
	// Allow only alphanumeric and underscores, starting with letter
	validName := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
	return validName.MatchString(name) && len(name) <= 63
}

// validateUsername checks if username is safe
func validateUsername(name string) bool {
	// Allow only alphanumeric and underscores
	validName := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	return validName.MatchString(name) && len(name) <= 63
}

// validateAdminIdentifier accepts the historical admin/email login values as
// well as a private site-owner username. The value is stored in the existing
// users.email column for compatibility, but it is never required to be a
// deliverable email address for the single-owner deployment mode.
func validateAdminIdentifier(identifier string) bool {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" || len(identifier) > 255 || strings.ContainsAny(identifier, "\r\n\t ") {
		return false
	}
	if identifier == defaultAdminLogin {
		return true
	}
	if _, err := mail.ParseAddress(identifier); err == nil {
		return true
	}
	return regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`).MatchString(identifier)
}

// validatePassword checks password strength
func validatePassword(password string) error {
	// 不对默认口令做白名单：任何路径（包括潜在绕过 validateOwnerAdmin 的
	// 调用方）都不得接受弱默认值 "admin"，交由下方的长度与复杂度检查兜底。
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	if len(password) > 128 {
		return fmt.Errorf("password must be at most 128 characters")
	}
	return nil
}

// validatePort checks if port is in valid range
func validatePort(port int) bool {
	return port > 0 && port <= 65535
}

// validateSSLMode checks if SSL mode is valid
func validateSSLMode(mode string) bool {
	validModes := map[string]bool{
		"disable": true, "require": true, "verify-ca": true, "verify-full": true,
	}
	return validModes[mode]
}

// TestDatabaseRequest represents database test request
type TestDatabaseRequest struct {
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port" binding:"required"`
	User     string `json:"user" binding:"required"`
	Password string `json:"password"`
	DBName   string `json:"dbname" binding:"required"`
	SSLMode  string `json:"sslmode"`
}

// testDatabase tests database connection
func testDatabase(c *gin.Context) {
	var req TestDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Security: Validate all inputs to prevent injection attacks
	addresses, err := resolveSetupTarget(req.Host, isLocalSetupRequest(c.Request) || privateSetupTargetsEnabled())
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid database target: "+err.Error())
		return
	}
	if !validatePort(req.Port) {
		response.Error(c, http.StatusBadRequest, "Invalid port number")
		return
	}
	if !validateUsername(req.User) {
		response.Error(c, http.StatusBadRequest, "Invalid username format")
		return
	}
	if !validateDBName(req.DBName) {
		response.Error(c, http.StatusBadRequest, "Invalid database name format")
		return
	}

	if req.SSLMode == "" {
		req.SSLMode = "disable"
	}
	if !validateSSLMode(req.SSLMode) {
		response.Error(c, http.StatusBadRequest, "Invalid SSL mode")
		return
	}

	cfg := &DatabaseConfig{
		Host:              req.Host,
		Port:              req.Port,
		User:              req.User,
		Password:          req.Password,
		DBName:            req.DBName,
		SSLMode:           req.SSLMode,
		resolvedAddresses: addresses,
	}

	if err := TestDatabaseConnection(cfg); err != nil {
		log.Printf("setup database connection test failed for %s:%d", cfg.Host, cfg.Port)
		response.Error(c, http.StatusBadRequest, "Database connection failed; verify the connection settings")
		return
	}

	response.Success(c, gin.H{"message": "Connection successful"})
}

// TestRedisRequest represents Redis test request
type TestRedisRequest struct {
	Host      string `json:"host" binding:"required"`
	Port      int    `json:"port" binding:"required"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	DB        int    `json:"db"`
	EnableTLS bool   `json:"enable_tls"`
}

// testRedis tests Redis connection
func testRedis(c *gin.Context) {
	var req TestRedisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Security: Validate inputs
	addresses, err := resolveSetupTarget(req.Host, isLocalSetupRequest(c.Request) || privateSetupTargetsEnabled())
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid Redis target: "+err.Error())
		return
	}
	if !validatePort(req.Port) {
		response.Error(c, http.StatusBadRequest, "Invalid port number")
		return
	}
	if req.DB < 0 || req.DB > 15 {
		response.Error(c, http.StatusBadRequest, "Invalid Redis database number (0-15)")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if len(req.Username) > 128 {
		response.Error(c, http.StatusBadRequest, "Invalid Redis username")
		return
	}

	cfg := &RedisConfig{
		Host:              req.Host,
		Port:              req.Port,
		Username:          req.Username,
		Password:          req.Password,
		DB:                req.DB,
		EnableTLS:         req.EnableTLS,
		resolvedAddresses: addresses,
	}

	if err := TestRedisConnection(cfg); err != nil {
		log.Printf("setup Redis connection test failed for %s:%d", cfg.Host, cfg.Port)
		response.Error(c, http.StatusBadRequest, "Redis connection failed; verify the connection settings")
		return
	}

	response.Success(c, gin.H{"message": "Connection successful"})
}

// InstallRequest represents installation request
type InstallRequest struct {
	Database DatabaseConfig `json:"database" binding:"required"`
	Redis    RedisConfig    `json:"redis" binding:"required"`
	Admin    AdminConfig    `json:"admin" binding:"required"`
	Server   ServerConfig   `json:"server"`
}

// install performs the installation
func install(c *gin.Context) {
	// TOCTOU Protection: Acquire mutex to prevent concurrent installation
	installMutex.Lock()
	defer installMutex.Unlock()

	// Double-check after acquiring lock
	if !NeedsSetup() {
		response.Error(c, http.StatusForbidden, "Setup is not allowed: system is already installed")
		return
	}

	var req InstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	req.Admin.Email = strings.TrimSpace(req.Admin.Email)
	req.Database.Host = strings.TrimSpace(req.Database.Host)
	req.Database.User = strings.TrimSpace(req.Database.User)
	req.Database.DBName = strings.TrimSpace(req.Database.DBName)
	req.Redis.Host = strings.TrimSpace(req.Redis.Host)
	req.Redis.Username = strings.TrimSpace(req.Redis.Username)

	// ========== COMPREHENSIVE INPUT VALIDATION ==========
	// Database validation
	allowPrivateTargets := isLocalSetupRequest(c.Request) || privateSetupTargetsEnabled()
	databaseAddresses, err := resolveSetupTarget(req.Database.Host, allowPrivateTargets)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid database target: "+err.Error())
		return
	}
	req.Database.resolvedAddresses = databaseAddresses
	if !validatePort(req.Database.Port) {
		response.Error(c, http.StatusBadRequest, "Invalid database port")
		return
	}
	if !validateUsername(req.Database.User) {
		response.Error(c, http.StatusBadRequest, "Invalid database username")
		return
	}
	if !validateDBName(req.Database.DBName) {
		response.Error(c, http.StatusBadRequest, "Invalid database name")
		return
	}

	// Redis validation
	redisAddresses, err := resolveSetupTarget(req.Redis.Host, allowPrivateTargets)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid Redis target: "+err.Error())
		return
	}
	req.Redis.resolvedAddresses = redisAddresses
	if !validatePort(req.Redis.Port) {
		response.Error(c, http.StatusBadRequest, "Invalid Redis port")
		return
	}
	if req.Redis.DB < 0 || req.Redis.DB > 15 {
		response.Error(c, http.StatusBadRequest, "Invalid Redis database number")
		return
	}
	if len(req.Redis.Username) > 128 {
		response.Error(c, http.StatusBadRequest, "Invalid Redis username")
		return
	}

	// Admin validation
	if err := validateOwnerAdmin(req.Admin); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// Server validation
	if req.Server.Port != 0 && !validatePort(req.Server.Port) {
		response.Error(c, http.StatusBadRequest, "Invalid server port")
		return
	}

	// ========== SET DEFAULTS ==========
	if req.Database.SSLMode == "" {
		req.Database.SSLMode = "disable"
	}
	if !validateSSLMode(req.Database.SSLMode) {
		response.Error(c, http.StatusBadRequest, "Invalid SSL mode")
		return
	}
	if req.Server.Host == "" {
		req.Server.Host = "0.0.0.0"
	}
	if req.Server.Port == 0 {
		req.Server.Port = 8080
	}
	if req.Server.Mode == "" {
		req.Server.Mode = "release"
	}
	// Validate server mode
	if req.Server.Mode != "release" && req.Server.Mode != "debug" {
		response.Error(c, http.StatusBadRequest, "Invalid server mode (must be 'release' or 'debug')")
		return
	}

	cfg := &SetupConfig{
		Database: req.Database,
		Redis:    req.Redis,
		Admin:    req.Admin,
		Server:   req.Server,
		JWT: JWTConfig{
			ExpireHour: 24,
		},
	}

	if err := Install(cfg); err != nil {
		log.Printf("setup installation failed")
		response.Error(c, http.StatusInternalServerError, "Installation failed; check the server logs for details")
		return
	}

	// Schedule service restart in background after sending response
	// This ensures the client receives the success response before the service restarts
	go func() {
		// Wait a moment to ensure the response is sent
		time.Sleep(500 * time.Millisecond)
		sysutil.RestartServiceAsync()
	}()

	response.Success(c, gin.H{
		"message": "Installation completed successfully. Service will restart automatically.",
		"restart": true,
	})
}

// AdminSetupRequest is intentionally limited to the site-owner credentials.
// Database, Redis and JWT values remain server-side environment/configuration
// data and are never accepted from the browser in automatic owner-setup mode.
type AdminSetupRequest struct {
	Email    string       `json:"email"`
	Password string       `json:"password"`
	Admin    *AdminConfig `json:"admin,omitempty"`
}

func completeAdminSetup(c *gin.Context) {
	installMutex.Lock()
	defer installMutex.Unlock()

	if !AdminSetupPending() {
		response.Error(c, http.StatusConflict, "Site owner setup is not pending")
		return
	}

	var req AdminSetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}
	admin := AdminConfig{Email: req.Email, Password: req.Password}
	if req.Admin != nil {
		if admin.Email == "" {
			admin.Email = req.Admin.Email
		}
		if admin.Password == "" {
			admin.Password = req.Admin.Password
		}
	}
	admin.Email = strings.TrimSpace(admin.Email)
	if err := validateOwnerAdmin(admin); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := CompleteAdminSetup(admin); err != nil {
		log.Printf("owner setup failed")
		if errors.Is(err, ErrAdminLoginAlreadyExists) {
			response.Error(c, http.StatusConflict, "That site-owner login is already in use; choose another login")
			return
		}
		response.Error(c, http.StatusInternalServerError, "Owner setup failed; check the server logs for details")
		return
	}

	// The setup server is a short-lived bootstrap process. On Linux the
	// supervisor/container restarts it into normal application mode.
	go func() {
		time.Sleep(500 * time.Millisecond)
		sysutil.RestartServiceAsync()
	}()

	response.Success(c, gin.H{
		"message": "Site owner setup completed successfully. Service will restart automatically.",
		"restart": true,
	})
}
