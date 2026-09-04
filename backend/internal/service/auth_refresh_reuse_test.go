//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// tombstoneRefreshTokenCache 在 refreshTokenCacheStub 的空实现基础上补齐
// 轮转墓碑能力，模拟 Redis 侧的 reuse-detection 行为。
type tombstoneRefreshTokenCache struct {
	refreshTokenCacheStub
	store     map[string]*RefreshTokenData
	families  map[string][]string // familyID -> token hashes
	tombstone map[string]string   // rotated token hash -> familyID
	familyIDs []string
}

func newTombstoneRefreshTokenCache() *tombstoneRefreshTokenCache {
	return &tombstoneRefreshTokenCache{
		store:     map[string]*RefreshTokenData{},
		families:  map[string][]string{},
		tombstone: map[string]string{},
	}
}

func (c *tombstoneRefreshTokenCache) StoreRefreshToken(_ context.Context, tokenHash string, data *RefreshTokenData, _ time.Duration) error {
	c.store[tokenHash] = data
	return nil
}

func (c *tombstoneRefreshTokenCache) GetRefreshToken(_ context.Context, tokenHash string) (*RefreshTokenData, error) {
	if data, ok := c.store[tokenHash]; ok {
		return data, nil
	}
	return nil, ErrRefreshTokenNotFound
}

func (c *tombstoneRefreshTokenCache) DeleteRefreshToken(_ context.Context, tokenHash string) error {
	delete(c.store, tokenHash)
	return nil
}

func (c *tombstoneRefreshTokenCache) DeleteTokenFamily(_ context.Context, familyID string) error {
	c.familyIDs = append(c.familyIDs, familyID)
	for _, hash := range c.families[familyID] {
		delete(c.store, hash)
	}
	delete(c.families, familyID)
	return nil
}

func (c *tombstoneRefreshTokenCache) AddToFamilyTokenSet(_ context.Context, familyID, tokenHash string, _ time.Duration) error {
	c.families[familyID] = append(c.families[familyID], tokenHash)
	return nil
}

func (c *tombstoneRefreshTokenCache) MarkRefreshTokenRotated(_ context.Context, tokenHash, familyID string, _ time.Duration) error {
	c.tombstone[tokenHash] = familyID
	return nil
}

func (c *tombstoneRefreshTokenCache) RotatedRefreshTokenFamily(_ context.Context, tokenHash string) (string, error) {
	return c.tombstone[tokenHash], nil
}

// 已轮转的旧 Refresh Token 再次提交时，整个家族必须被撤销：
// 之后即使持有轮转后的新 Token 也无法继续刷新。
func TestRefreshTokenReuseRevokesFamily(t *testing.T) {
	user := &User{ID: 1, Email: "reuse@example.com", Role: RoleUser, Status: StatusActive}
	userRepo := &userRepoStub{user: user}
	cache := newTombstoneRefreshTokenCache()
	svc := newOAuthEmailFlowAuthService(userRepo, &redeemCodeRepoStub{}, cache, map[string]string{}, nil, nil)

	ctx := context.Background()
	pair1, err := svc.GenerateTokenPair(ctx, user, "family-1")
	require.NoError(t, err)
	require.NotEmpty(t, pair1.RefreshToken)

	// 正常轮转：旧 Token 失效，签发新 Token 对。
	pair2, err := svc.RefreshTokenPair(ctx, pair1.RefreshToken)
	require.NoError(t, err)
	require.NotEqual(t, pair1.RefreshToken, pair2.RefreshToken)

	// 重放旧 Token：检测到 reuse，撤销整个家族。
	_, err = svc.RefreshTokenPair(ctx, pair1.RefreshToken)
	require.ErrorIs(t, err, ErrRefreshTokenInvalid)
	require.Equal(t, []string{"family-1"}, cache.familyIDs, "replayed token must revoke its whole family")

	// 家族撤销后，轮转产生的新 Token 同样失效。
	_, err = svc.RefreshTokenPair(ctx, pair2.RefreshToken)
	require.ErrorIs(t, err, ErrRefreshTokenInvalid)
}

// noTombstoneRefreshTokenCache 支持存储但未实现墓碑扩展，模拟旧缓存实现：
// 重放检测应优雅降级——只拒绝被重放的 Token，不影响家族内新 Token。
type noTombstoneRefreshTokenCache struct {
	refreshTokenCacheStub
	store    map[string]*RefreshTokenData
	families map[string][]string
}

func newNoTombstoneRefreshTokenCache() *noTombstoneRefreshTokenCache {
	return &noTombstoneRefreshTokenCache{
		store:    map[string]*RefreshTokenData{},
		families: map[string][]string{},
	}
}

func (c *noTombstoneRefreshTokenCache) StoreRefreshToken(_ context.Context, tokenHash string, data *RefreshTokenData, _ time.Duration) error {
	c.store[tokenHash] = data
	return nil
}

func (c *noTombstoneRefreshTokenCache) GetRefreshToken(_ context.Context, tokenHash string) (*RefreshTokenData, error) {
	if data, ok := c.store[tokenHash]; ok {
		return data, nil
	}
	return nil, ErrRefreshTokenNotFound
}

func (c *noTombstoneRefreshTokenCache) DeleteRefreshToken(_ context.Context, tokenHash string) error {
	delete(c.store, tokenHash)
	return nil
}

func (c *noTombstoneRefreshTokenCache) AddToUserTokenSet(_ context.Context, _ int64, _ string, _ time.Duration) error {
	return nil
}

func (c *noTombstoneRefreshTokenCache) AddToFamilyTokenSet(_ context.Context, familyID, tokenHash string, _ time.Duration) error {
	c.families[familyID] = append(c.families[familyID], tokenHash)
	return nil
}

func TestRefreshTokenReuseWithoutTombstoneSupportStillRejectsToken(t *testing.T) {
	user := &User{ID: 1, Email: "reuse2@example.com", Role: RoleUser, Status: StatusActive}
	userRepo := &userRepoStub{user: user}
	cache := newNoTombstoneRefreshTokenCache()
	svc := newOAuthEmailFlowAuthService(userRepo, &redeemCodeRepoStub{}, cache, map[string]string{}, nil, nil)

	ctx := context.Background()
	pair1, err := svc.GenerateTokenPair(ctx, user, "family-1")
	require.NoError(t, err)

	pair2, err := svc.RefreshTokenPair(ctx, pair1.RefreshToken)
	require.NoError(t, err)

	// 无墓碑支持时，重放只被当作"Token 不存在"拒绝，新 Token 不受影响。
	_, err = svc.RefreshTokenPair(ctx, pair1.RefreshToken)
	require.ErrorIs(t, err, ErrRefreshTokenInvalid)

	_, err = svc.RefreshTokenPair(ctx, pair2.RefreshToken)
	require.NoError(t, err)
}
