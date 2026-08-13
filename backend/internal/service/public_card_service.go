package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	infraerrors "github.com/yuchen0x1/sub3api-relay/internal/pkg/errors"
)

var (
	ErrPublicCardInvalid     = infraerrors.BadRequest("PUBLIC_CARD_INVALID", "invalid card code")
	ErrPublicCardUnsupported = infraerrors.BadRequest("PUBLIC_CARD_UNSUPPORTED", "only positive balance cards can be redeemed publicly")
	ErrPublicCardKeyInvalid  = infraerrors.BadRequest("PUBLIC_CARD_KEY_INVALID", "invalid api key")
)

type PublicCardRedeemRequest struct {
	Code string `json:"code"`
}

type PublicCardKeyQueryRequest struct {
	APIKey string `json:"api_key"`
}

type PublicCardKeyBalance struct {
	KeyPreview     string     `json:"key_preview"`
	Status         string     `json:"status"`
	Quota          float64    `json:"quota"`
	QuotaUsed      float64    `json:"quota_used"`
	QuotaRemaining float64    `json:"quota_remaining"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type PublicCardRedeemResult struct {
	APIKey string `json:"api_key"`
	PublicCardKeyBalance
}

type PublicCardService struct {
	redeemService *RedeemService
	apiKeyService *APIKeyService
	userRepo      UserRepository
}

func NewPublicCardService(
	redeemService *RedeemService,
	apiKeyService *APIKeyService,
	userRepo UserRepository,
) *PublicCardService {
	return &PublicCardService{
		redeemService: redeemService,
		apiKeyService: apiKeyService,
		userRepo:      userRepo,
	}
}

func (s *PublicCardService) Redeem(ctx context.Context, req PublicCardRedeemRequest) (*PublicCardRedeemResult, error) {
	if s == nil || s.redeemService == nil || s.apiKeyService == nil || s.userRepo == nil {
		return nil, infraerrors.New(500, "PUBLIC_CARD_SERVICE_UNAVAILABLE", "card redemption is temporarily unavailable")
	}

	code := normalizePublicCardCode(req.Code)
	if code == "" {
		return nil, ErrPublicCardInvalid
	}

	card, err := s.redeemService.GetByCode(ctx, code)
	if err != nil {
		if errors.Is(err, ErrRedeemCodeNotFound) {
			return nil, ErrPublicCardInvalid
		}
		return nil, fmt.Errorf("get redeem code: %w", err)
	}
	if err := validatePublicBalanceCard(card); err != nil {
		return nil, err
	}

	user, err := s.createPromoUser(ctx, card)
	if err != nil {
		return nil, fmt.Errorf("create promo user: %w", err)
	}

	redeemed, err := s.redeemService.Redeem(ContextSkipRedeemAffiliate(ctx), user.ID, code)
	if err != nil {
		return nil, fmt.Errorf("redeem public card: %w", err)
	}

	key, err := s.createRedeemedAPIKey(ctx, user.ID, redeemed)
	if err != nil {
		return nil, fmt.Errorf("create redeemed api key: %w", err)
	}

	return &PublicCardRedeemResult{
		APIKey:               key.Key,
		PublicCardKeyBalance: publicCardBalanceFromAPIKey(key),
	}, nil
}

func (s *PublicCardService) QueryKey(ctx context.Context, req PublicCardKeyQueryRequest) (*PublicCardKeyBalance, error) {
	if s == nil || s.apiKeyService == nil {
		return nil, infraerrors.New(500, "PUBLIC_CARD_SERVICE_UNAVAILABLE", "card key query is temporarily unavailable")
	}

	apiKey := strings.TrimSpace(req.APIKey)
	if len(apiKey) < 16 {
		return nil, ErrPublicCardKeyInvalid
	}

	key, err := s.apiKeyService.GetByKey(ctx, apiKey)
	if err != nil {
		if errors.Is(err, ErrAPIKeyNotFound) {
			return nil, ErrPublicCardKeyInvalid
		}
		return nil, fmt.Errorf("get api key: %w", err)
	}

	return &PublicCardKeyBalance{
		KeyPreview:     maskPublicAPIKey(apiKey),
		Status:         publicAPIKeyStatus(key),
		Quota:          key.Quota,
		QuotaUsed:      key.QuotaUsed,
		QuotaRemaining: key.GetQuotaRemaining(),
		ExpiresAt:      key.ExpiresAt,
		LastUsedAt:     key.LastUsedAt,
		CreatedAt:      key.CreatedAt,
		UpdatedAt:      key.UpdatedAt,
	}, nil
}

func (s *PublicCardService) createPromoUser(ctx context.Context, card *RedeemCode) (*User, error) {
	suffix, err := randomPublicCardHex(12)
	if err != nil {
		return nil, err
	}
	password, err := randomPublicCardHex(32)
	if err != nil {
		return nil, err
	}

	user := &User{
		Email:        fmt.Sprintf("promo+%s@sub3api.invalid", suffix),
		Username:     "Promo Card User",
		Notes:        fmt.Sprintf("Created by public card redemption: %s", card.Code),
		Role:         RoleUser,
		Status:       StatusActive,
		Concurrency:  5,
		SignupSource: "email",
	}
	if card.GroupID != nil {
		user.AllowedGroups = []int64{*card.GroupID}
	}
	if err := user.SetPassword(password); err != nil {
		return nil, err
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *PublicCardService) createRedeemedAPIKey(ctx context.Context, userID int64, card *RedeemCode) (*APIKey, error) {
	req := CreateAPIKeyRequest{
		Name:    fmt.Sprintf("Promo card %s", shortCardCode(card.Code)),
		GroupID: card.GroupID,
		Quota:   card.Value,
	}
	key, err := s.apiKeyService.Create(ctx, userID, req)
	if err == nil {
		return key, nil
	}
	if req.GroupID != nil && errors.Is(err, ErrGroupNotAllowed) {
		req.GroupID = nil
		return s.apiKeyService.Create(ctx, userID, req)
	}
	return nil, err
}

func validatePublicBalanceCard(card *RedeemCode) error {
	if card == nil {
		return ErrPublicCardInvalid
	}
	if card.IsExpired() {
		return ErrRedeemCodeExpired
	}
	if !card.CanUse() {
		return ErrRedeemCodeUsed
	}
	if card.Type != RedeemTypeBalance || card.Value <= 0 {
		return ErrPublicCardUnsupported
	}
	return nil
}

func normalizePublicCardCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func publicCardBalanceFromAPIKey(key *APIKey) PublicCardKeyBalance {
	if key == nil {
		return PublicCardKeyBalance{}
	}
	return PublicCardKeyBalance{
		KeyPreview:     maskPublicAPIKey(key.Key),
		Status:         publicAPIKeyStatus(key),
		Quota:          key.Quota,
		QuotaUsed:      key.QuotaUsed,
		QuotaRemaining: key.GetQuotaRemaining(),
		ExpiresAt:      key.ExpiresAt,
		LastUsedAt:     key.LastUsedAt,
		CreatedAt:      key.CreatedAt,
		UpdatedAt:      key.UpdatedAt,
	}
}

func publicAPIKeyStatus(key *APIKey) string {
	if key == nil {
		return StatusAPIKeyDisabled
	}
	if key.IsExpired() {
		return StatusAPIKeyExpired
	}
	if key.IsQuotaExhausted() {
		return StatusAPIKeyQuotaExhausted
	}
	return key.Status
}

func maskPublicAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 10 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-6:]
}

func shortCardCode(code string) string {
	code = strings.ReplaceAll(strings.TrimSpace(code), "-", "")
	if len(code) <= 8 {
		return code
	}
	return code[len(code)-8:]
}

func randomPublicCardHex(bytesLen int) (string, error) {
	b := make([]byte, bytesLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
