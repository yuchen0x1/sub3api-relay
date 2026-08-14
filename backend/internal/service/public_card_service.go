package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	dbent "github.com/yuchen0x1/sub3api-relay/ent"
	"github.com/yuchen0x1/sub3api-relay/ent/redeemcode"
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
	Concurrency    int        `json:"concurrency"`
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

	key, err := s.redeemPublicCardAtomically(ctx, card)
	if err != nil {
		return nil, fmt.Errorf("redeem public card: %w", err)
	}

	balance := publicCardBalanceFromAPIKey(key)
	balance.Concurrency = card.PromoConcurrency()
	return &PublicCardRedeemResult{
		APIKey:               key.Key,
		PublicCardKeyBalance: balance,
	}, nil
}

func (s *PublicCardService) QueryKey(ctx context.Context, req PublicCardKeyQueryRequest) (*PublicCardKeyBalance, error) {
	if s == nil || s.apiKeyService == nil || s.redeemService == nil {
		return nil, infraerrors.New(500, "PUBLIC_CARD_SERVICE_UNAVAILABLE", "card key query is temporarily unavailable")
	}

	credential := strings.TrimSpace(req.APIKey)
	if len(credential) < 3 {
		return nil, ErrPublicCardKeyInvalid
	}

	key, err := s.apiKeyService.GetByKeyFresh(ctx, credential)
	if err == nil {
		balance := publicCardBalanceFromAPIKey(key)
		balance.KeyPreview = maskPublicAPIKey(credential)
		return &balance, nil
	}
	if !errors.Is(err, ErrAPIKeyNotFound) {
		return nil, fmt.Errorf("get api key: %w", err)
	}

	card, err := s.redeemService.GetByCode(ctx, credential)
	if err != nil {
		if errors.Is(err, ErrRedeemCodeNotFound) {
			return nil, ErrPublicCardKeyInvalid
		}
		return nil, fmt.Errorf("get redeem code: %w", err)
	}
	if card.Type != RedeemTypeBalance && card.Type != RedeemTypePromo {
		return nil, ErrPublicCardKeyInvalid
	}
	if card.APIKey != nil {
		balance := publicCardBalanceFromAPIKey(card.APIKey)
		balance.Concurrency = card.PromoConcurrency()
		return &balance, nil
	}
	if !card.CanUse() && !card.IsExpired() {
		return nil, ErrPublicCardKeyInvalid
	}

	status := card.Status
	remaining := card.Value
	if card.IsExpired() {
		status = StatusExpired
		remaining = 0
	}
	return &PublicCardKeyBalance{
		Status:         status,
		Concurrency:    card.PromoConcurrency(),
		Quota:          card.Value,
		QuotaRemaining: remaining,
		CreatedAt:      card.CreatedAt,
		UpdatedAt:      card.CreatedAt,
	}, nil
}

func (s *PublicCardService) redeemPublicCardAtomically(ctx context.Context, card *RedeemCode) (*APIKey, error) {
	if s.redeemService.entClient == nil {
		return nil, infraerrors.New(500, "PUBLIC_CARD_SERVICE_UNAVAILABLE", "card redemption is temporarily unavailable")
	}

	user, err := newPromoCardUser(card)
	if err != nil {
		return nil, fmt.Errorf("build promo user: %w", err)
	}
	keyValue, err := s.apiKeyService.GenerateKey()
	if err != nil {
		return nil, err
	}

	tx, err := s.redeemService.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)

	user.Balance = card.Value
	if err := s.userRepo.Create(txCtx, user); err != nil {
		return nil, fmt.Errorf("create promo user: %w", err)
	}

	keyBuilder := tx.APIKey.Create().
		SetUserID(user.ID).
		SetKey(keyValue).
		SetName(fmt.Sprintf("Promo card %s", shortCardCode(card.Code))).
		SetStatus(StatusAPIKeyActive).
		SetQuota(card.Value).
		SetQuotaUsed(0)
	if card.GroupID != nil {
		keyBuilder.SetGroupID(*card.GroupID)
	}
	createdKey, err := keyBuilder.Save(txCtx)
	if err != nil {
		return nil, fmt.Errorf("create redeemed api key: %w", err)
	}

	usedAt := time.Now().UTC()
	updated, err := tx.RedeemCode.Update().
		Where(
			redeemcode.IDEQ(card.ID),
			redeemcode.StatusEQ(StatusUnused),
			redeemcode.Or(redeemcode.ExpiresAtIsNil(), redeemcode.ExpiresAtGT(usedAt)),
		).
		SetStatus(StatusUsed).
		SetUsedBy(user.ID).
		SetUsedAt(usedAt).
		Save(txCtx)
	if err != nil {
		return nil, fmt.Errorf("mark card as used: %w", err)
	}
	if updated != 1 {
		if card.IsExpiredAt(usedAt) {
			return nil, ErrRedeemCodeExpired
		}
		return nil, ErrRedeemCodeUsed
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit public card redemption: %w", err)
	}

	return &APIKey{
		ID:        createdKey.ID,
		UserID:    user.ID,
		Key:       createdKey.Key,
		Name:      createdKey.Name,
		GroupID:   createdKey.GroupID,
		Status:    createdKey.Status,
		Quota:     createdKey.Quota,
		QuotaUsed: createdKey.QuotaUsed,
		CreatedAt: createdKey.CreatedAt,
		UpdatedAt: createdKey.UpdatedAt,
		User:      user,
	}, nil
}

func newPromoCardUser(card *RedeemCode) (*User, error) {
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
		Concurrency:  card.PromoConcurrency(),
		SignupSource: "email",
	}
	if card.GroupID != nil {
		user.AllowedGroups = []int64{*card.GroupID}
	}
	if err := user.SetPassword(password); err != nil {
		return nil, err
	}
	return user, nil
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
	if (card.Type != RedeemTypeBalance && card.Type != RedeemTypePromo) || card.Value <= 0 {
		return ErrPublicCardUnsupported
	}
	return nil
}

func normalizePublicCardCode(code string) string {
	return strings.TrimSpace(code)
}

func publicCardBalanceFromAPIKey(key *APIKey) PublicCardKeyBalance {
	if key == nil {
		return PublicCardKeyBalance{}
	}
	concurrency := 0
	if key.User != nil {
		concurrency = key.User.Concurrency
	}
	return PublicCardKeyBalance{
		KeyPreview:     maskPublicAPIKey(key.Key),
		Status:         publicAPIKeyStatus(key),
		Concurrency:    concurrency,
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
