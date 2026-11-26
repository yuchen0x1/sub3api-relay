package repository

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/config"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

const testEncryptionKeyHex = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func mustPrepareCredentials(t *testing.T, cipher *credentialCipher, credentials map[string]any) map[string]any {
	t.Helper()
	stored, err := cipher.prepareForWrite(credentials)
	require.NoError(t, err)
	return stored
}

type failingCredentialEncryptor struct {
	err error
}

func (e failingCredentialEncryptor) Encrypt(string) (string, error) { return "", e.err }
func (e failingCredentialEncryptor) Decrypt(string) (string, error) { return "", e.err }

func TestCredentialCipherEncryptionFailureBlocksWrites(t *testing.T) {
	failure := errors.New("encryption failed")
	cipher := &credentialCipher{encryptor: failingCredentialEncryptor{err: failure}}
	credentials := map[string]any{"api_key": "sk-test-secret", "base_url": "https://example.com"}

	stored, err := cipher.prepareForWrite(credentials)
	require.ErrorIs(t, err, failure)
	require.Nil(t, stored, "a partially encrypted document must never be returned")
	require.Equal(t, "sk-test-secret", credentials["api_key"], "the caller's input must not be mutated")

	doc, digest, keyDigest, err := prepareCredentialsStorage(cipher, credentials)
	require.ErrorIs(t, err, failure)
	require.Nil(t, doc)
	require.Nil(t, digest)
	require.Nil(t, keyDigest)

	// Nil clients make any attempt to reach SQL before rejecting encryption fail.
	repo := &accountRepository{credCipher: cipher}
	require.ErrorIs(t, repo.UpdateCredentials(context.Background(), 1, credentials), failure)
	count, err := repo.BulkUpdate(context.Background(), []int64{1}, service.AccountBulkUpdate{Credentials: credentials})
	require.ErrorIs(t, err, failure)
	require.Zero(t, count)
}

func TestCredentialCipherInvalidConfiguredKeyBlocksWrites(t *testing.T) {
	cfg := &config.Config{}
	cfg.Totp.EncryptionKeyConfigured = true
	cfg.Totp.EncryptionKey = "invalid-key"
	cipher := NewAccountCredentialCipher(cfg)
	require.Error(t, cipher.initErr)
	stored, err := cipher.prepareForWrite(map[string]any{"api_key": "sk-test-secret"})
	require.Error(t, err)
	require.Nil(t, stored)
}

func TestCredentialCipherMarshalFailureBlocksWrites(t *testing.T) {
	stored, err := testCipher(t).prepareForWrite(map[string]any{"api_key": make(chan int)})
	require.Error(t, err)
	require.Nil(t, stored)
}

func testCipher(t *testing.T) *credentialCipher {
	t.Helper()
	cfg := &config.Config{}
	cfg.Totp.EncryptionKey = testEncryptionKeyHex
	cfg.Totp.EncryptionKeyConfigured = true
	encryptor, err := NewAESEncryptor(cfg)
	if err != nil {
		t.Fatalf("NewAESEncryptor: %v", err)
	}
	return &credentialCipher{encryptor: encryptor}
}

func jsonEqual(t *testing.T, a, b any) bool {
	t.Helper()
	ab, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal a: %v", err)
	}
	bb, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal b: %v", err)
	}
	return string(ab) == string(bb)
}

// 加密开关关闭时必须保持明文直通（随机临时密钥不可用于持久化）。
func TestCredentialCipherPassthroughWhenDisabled(t *testing.T) {
	cipher := &credentialCipher{}
	in := map[string]any{"api_key": "sk-test-secret", "base_url": "https://x.example.com"}
	out := mustPrepareCredentials(t, cipher, in)
	if out["api_key"] != "sk-test-secret" {
		t.Fatalf("expected plaintext passthrough, got %v", out["api_key"])
	}
	if !jsonEqual(t, cipher.restoreForRead(out), in) {
		t.Fatal("restoreForRead should be identity when disabled")
	}
}

// 完整往返：敏感子键加密、非敏感子键与结构保持不变、读取还原。
func TestCredentialCipherRoundtrip(t *testing.T) {
	cipher := testCipher(t)
	in := map[string]any{
		"api_key":              "sk-ant-api03-supersecret",
		"access_token":         "oauth-access-value",
		"refresh_token":        "oauth-refresh-value",
		"cookie":               map[string]any{"session_key": "abc", "nested": []any{1, "two"}},
		"service_account_json": map[string]any{"private_key": "-----BEGIN"},
		"model_mapping":        map[string]any{"gpt-x": "gpt-y"},
		"base_url":             "https://relay.example.com",
		"expires_at":           "2099-01-01T00:00:00Z",
	}
	stored := mustPrepareCredentials(t, cipher, in)

	for _, key := range []string{"api_key", "access_token", "refresh_token", "cookie", "service_account_json"} {
		v, ok := stored[key].(string)
		if !ok || !strings.HasPrefix(v, credentialEnvelopePrefix) {
			t.Fatalf("sensitive key %q not enveloped: %v", key, stored[key])
		}
	}
	if stored["base_url"] != in["base_url"] || stored["expires_at"] != in["expires_at"] {
		t.Fatal("non-sensitive keys must stay plaintext")
	}
	if !jsonEqual(t, stored["model_mapping"], in["model_mapping"]) {
		t.Fatal("model_mapping must stay intact")
	}

	restored := cipher.restoreForRead(stored)
	for _, key := range []string{"api_key", "access_token", "refresh_token", "cookie", "service_account_json", "model_mapping", "base_url", "expires_at"} {
		if !jsonEqual(t, restored[key], in[key]) {
			t.Fatalf("roundtrip mismatch for %q: got %v want %v", key, restored[key], in[key])
		}
	}
}

// 幂等：已带信封的值不得二次加密。
func TestCredentialCipherIdempotent(t *testing.T) {
	cipher := testCipher(t)
	first := mustPrepareCredentials(t, cipher, map[string]any{"api_key": "sk-once"})
	enveloped, ok := first["api_key"].(string)
	if !ok {
		t.Fatal("expected an encrypted string value")
	}
	second := mustPrepareCredentials(t, cipher, first)
	secondValue, ok := second["api_key"].(string)
	if !ok {
		t.Fatal("expected an encrypted string value")
	}
	if secondValue != enveloped {
		t.Fatal("already enveloped value must not be re-encrypted")
	}
}

// 错误密钥解密必须保留密文原文（显式失败而非注入垃圾值），且不 panic。
func TestCredentialCipherWrongKeyKeepsCiphertext(t *testing.T) {
	encryptCipher := testCipher(t)
	stored := mustPrepareCredentials(t, encryptCipher, map[string]any{"api_key": "sk-value"})

	other := &config.Config{}
	other.Totp.EncryptionKey = strings.Repeat("ab", 32)
	other.Totp.EncryptionKeyConfigured = true
	otherEncryptor, err := NewAESEncryptor(other)
	if err != nil {
		t.Fatalf("NewAESEncryptor: %v", err)
	}
	decryptCipher := &credentialCipher{encryptor: otherEncryptor}

	restored := decryptCipher.restoreForRead(stored)
	if restored["api_key"] != stored["api_key"] {
		t.Fatal("undecryptable value must be preserved as ciphertext")
	}
}

// NewAccountCredentialCipher 闸门：密钥未显式配置时禁用加密。
func TestNewAccountCredentialCipherGate(t *testing.T) {
	missing := NewAccountCredentialCipher(nil)
	require.Nil(t, missing.encryptor)
	require.Error(t, missing.initErr, "missing configuration must block credential writes")
	_, _, _, err := prepareCredentialsStorage(missing, map[string]any{"api_key": "fixture-key"})
	require.Error(t, err)
	cfg := &config.Config{}
	cfg.Totp.EncryptionKey = testEncryptionKeyHex
	cfg.Totp.EncryptionKeyConfigured = false
	cfg.JWT.Secret = "stable-jwt-secret-at-least-32-bytes!!"
	if c := NewAccountCredentialCipher(cfg); c.encryptor != nil {
		t.Fatal("unconfigured key must yield passthrough cipher")
	}
	cfg.Totp.EncryptionKeyConfigured = true
	if c := NewAccountCredentialCipher(cfg); c.encryptor == nil {
		t.Fatal("configured key must enable encryption")
	}
}

func TestCredentialDigestKeyUsesStableJWTSecretWithoutTOTPEncryption(t *testing.T) {
	cfg := &config.Config{}
	cfg.JWT.Secret = "persisted-jwt-secret-from-security-secrets"
	first := NewAccountCredentialCipher(cfg)
	require.Nil(t, first.encryptor, "credential encryption remains opt-in")
	require.NotEmpty(t, first.hmacKey, "digest protection must not depend on TOTP encryption being enabled")

	credentials := map[string]any{"api_key": "sk-low-entropy"}
	_, firstDigest, _, err := prepareCredentialsStorage(first, credentials)
	require.NoError(t, err)
	_, legacyDigest, _, err := canonicalCredentialsPayload(credentials)
	require.NoError(t, err)
	require.NotEqual(t, legacyDigest, *firstDigest)

	restarted := NewAccountCredentialCipher(cfg)
	_, restartedDigest, _, err := prepareCredentialsStorage(restarted, credentials)
	require.NoError(t, err)
	require.Equal(t, *firstDigest, *restartedDigest, "the database-persisted JWT master yields stable digests across restarts")

	rotated := &config.Config{}
	rotated.JWT.Secret = "different-jwt-secret-after-coordinated-migration"
	other := NewAccountCredentialCipher(rotated)
	require.NotEqual(t, first.hmacKey, other.hmacKey, "JWT secret rotation changes the digest key and therefore requires a coordinated migration")
}

func TestAccountCredentialWritesFailClosedWithoutDigestKey(t *testing.T) {
	cipher := NewAccountCredentialCipher(&config.Config{})

	require.Error(t, cipher.initErr)
	_, _, _, err := prepareCredentialsStorage(cipher, map[string]any{"api_key": "low-entropy"})
	require.Error(t, err)
}

func TestPublicAccountRepositoriesRejectMissingCredentialCipher(t *testing.T) {
	repositories := map[string]*accountRepository{
		"account": NewAccountRepository(nil, nil, nil, nil).(*accountRepository),
		"admin":   NewAdminAccountRepository(nil, nil, nil, nil).(*accountRepository),
	}
	for name, repo := range repositories {
		t.Run(name, func(t *testing.T) {
			credentials := map[string]any{"api_key": "fixture-key"}
			require.NotNil(t, repo.credCipher)
			require.Error(t, repo.credCipher.initErr)
			require.Error(t, repo.Create(context.Background(), &service.Account{Credentials: credentials}))
			require.Error(t, repo.UpdateCredentials(context.Background(), 1, credentials))
			count, err := repo.BulkUpdate(context.Background(), []int64{1}, service.AccountBulkUpdate{Credentials: credentials})
			require.Error(t, err)
			require.Zero(t, count)
		})
	}
}

func TestCredentialDigestKeyUsesExternalEncryptionKeyWhenEnabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.JWT.Secret = "database-persisted-secret-not-independent-of-db-leak"
	cfg.Totp.EncryptionKeyConfigured = true
	cfg.Totp.EncryptionKey = testEncryptionKeyHex
	first := NewAccountCredentialCipher(cfg)
	require.NoError(t, first.initErr)
	require.NotEqual(t, deriveCredentialDigestKey(cfg.JWT.Secret), first.hmacKey)

	cfg.JWT.Secret = "a-different-database-persisted-secret"
	require.Equal(t, first.hmacKey, NewAccountCredentialCipher(cfg).hmacKey)
	cfg.Totp.EncryptionKey = strings.Repeat("ab", 32)
	require.NotEqual(t, first.hmacKey, NewAccountCredentialCipher(cfg).hmacKey)
}

// 摘要：确定性、与键序无关、api_key 摘要按存在性返回。
func TestCanonicalCredentialsPayloadDigest(t *testing.T) {
	a := map[string]any{"api_key": "sk-1", "access_token": "tok"}
	b := map[string]any{"access_token": "tok", "api_key": "sk-1"}
	_, digestA, akA, err := canonicalCredentialsPayload(a)
	if err != nil {
		t.Fatalf("canonicalCredentialsPayload: %v", err)
	}
	_, digestB, akB, err := canonicalCredentialsPayload(b)
	if err != nil {
		t.Fatalf("canonicalCredentialsPayload: %v", err)
	}
	if digestA == "" || digestA != digestB {
		t.Fatal("digest must be deterministic and key-order independent")
	}
	if akA == nil || akB == nil || *akA != *akB {
		t.Fatal("api_key digest must match for equal api_key values")
	}
	if len(*akA) != 64 {
		t.Fatalf("api_key digest must be sha256 hex (64 chars), got %d", len(*akA))
	}
	_, _, noAK, err := canonicalCredentialsPayload(map[string]any{"refresh_token": "r"})
	if err != nil {
		t.Fatalf("canonicalCredentialsPayload: %v", err)
	}
	if noAK != nil {
		t.Fatal("api_key digest must be nil when api_key absent")
	}
}

// 快照 JSON 摘要与 map 摘要一致（CAS 绑定一致性）。
func TestCredentialsDigestFromJSON(t *testing.T) {
	creds := map[string]any{"api_key": "sk-1", "access_token": "tok", "_token_version": 3}
	_, digest, _, err := canonicalCredentialsPayload(creds)
	if err != nil {
		t.Fatalf("canonicalCredentialsPayload: %v", err)
	}
	fromJSON := credentialsDigestFromJSON(`{"_token_version":3,"access_token":"tok","api_key":"sk-1"}`)
	if fromJSON == nil || *fromJSON != digest {
		t.Fatalf("digest from JSON mismatch: %v vs %s", fromJSON, digest)
	}
	if credentialsDigestFromJSON("not-json") != nil {
		t.Fatal("invalid JSON must yield nil digest")
	}
}

// 加密后的存储文档的摘要必须等于明文文档的摘要（写路径一致性核心不变量）。
func TestDigestStableUnderEncryption(t *testing.T) {
	cipher := testCipher(t)
	creds := map[string]any{"api_key": "sk-1", "access_token": "tok", "base_url": "https://x"}
	_, plainDigest, plainAK, err := canonicalCredentialsPayload(creds)
	if err != nil {
		t.Fatalf("canonicalCredentialsPayload: %v", err)
	}
	stored, storedDigest, storedAK, err := prepareCredentialsStorage(cipher, creds)
	if err != nil {
		t.Fatalf("prepareCredentialsStorage: %v", err)
	}
	if plainDigest != *storedDigest || *plainAK != *storedAK {
		t.Fatal("digests must be computed over plaintext, independent of encryption")
	}
	if stored["api_key"] == "sk-1" {
		t.Fatal("stored doc must be encrypted")
	}
	// 明文直通时摘要仍应写入（为后续启用加密预留一致性）。
	_, plainStoredDigest, _, err := prepareCredentialsStorage(nil, creds)
	if err != nil {
		t.Fatalf("prepareCredentialsStorage: %v", err)
	}
	if plainStoredDigest == nil || *plainStoredDigest != plainDigest {
		t.Fatal("passthrough mode must still persist the plaintext digest")
	}
}

func TestCredentialDigestCandidatesAreKeyedVersionedAndLegacyCompatible(t *testing.T) {
	credentials := map[string]any{"api_key": "sk-low-entropy", "access_token": "token"}
	key := []byte("stable-encryption-key-for-tests-32b")
	payload, primary, legacy, apiPrimary, apiLegacy, err := credentialDigestCandidates(credentials, key)
	require.NoError(t, err)
	require.Len(t, primary, 64)
	require.NotEqual(t, legacy, primary)
	require.NotNil(t, apiPrimary)
	require.NotNil(t, apiLegacy)
	require.NotEqual(t, *apiLegacy, *apiPrimary)
	require.Equal(t, primary, credentialHMAC(key, credentialDigestDomain, payload))
	require.NotEqual(t, primary, credentialHMAC(key, apiKeyDigestDomain, payload), "document and API-key digests must be domain-separated")

	_, repeated, _, repeatedAPIKey, _, err := credentialDigestCandidates(credentials, key)
	require.NoError(t, err)
	require.Equal(t, primary, repeated)
	require.Equal(t, apiPrimary, repeatedAPIKey)

	_, noKeyDigest, _, _, _, err := credentialDigestCandidates(credentials, nil)
	require.NoError(t, err)
	require.Equal(t, legacy, noKeyDigest, "legacy writes without a configured stable key must remain compatible")
	require.Equal(t, credentialDigestVersionHMAC, credentialDigestVersionForKey(key))
	require.Equal(t, credentialDigestVersionLegacy, credentialDigestVersionForKey(nil))
}

func TestCredentialDigestClausesFailClosedForUnknownVersions(t *testing.T) {
	equal := credentialsEqualClause("a.credentials", "a.credentials_digest", "a.credentials_digest_version", "$1", "$2", "$3")
	require.Contains(t, equal, "a.credentials_digest_version = 2 THEN a.credentials_digest = $2")
	require.Contains(t, equal, "a.credentials_digest_version = 1 OR a.credentials_digest_version IS NULL")
	require.Contains(t, equal, "ELSE FALSE END")

	changed := credentialsChangedClause("a.credentials", "a.credentials_digest", "a.credentials_digest_version", "$1", "$2", "$3")
	require.Contains(t, changed, "ELSE TRUE END")
	apiKeyEqual := credentialsAPIKeyEqualClause("a.credentials", "a.credentials_api_key_digest", "a.credentials_digest_version", "$1", "$2", "$3")
	require.Contains(t, apiKeyEqual, "a.credentials_digest_version = 2 THEN a.credentials_api_key_digest IS NOT DISTINCT FROM $2")
	require.Contains(t, apiKeyEqual, "ELSE FALSE END")
}

func TestBackfillAccountCredentialDigestsUsesCASAndSkipsUndecryptableRows(t *testing.T) {
	raw := []byte(`{"api_key":"sk-legacy"}`)
	plain := map[string]any{"api_key": "sk-legacy"}
	_, hmacDigest, legacyDigest, hmacAPIKeyDigest, legacyAPIKeyDigest, err := credentialDigestCandidates(plain, []byte("stable-test-hmac-key"))
	require.NoError(t, err)

	t.Run("legacy row is upgraded only when the observed credentials and digests match", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })
		mock.ExpectQuery(`(?s)SELECT id, credentials, credentials_digest_version, credentials_digest, credentials_api_key_digest.*WHERE id > \$1.*ORDER BY id.*LIMIT \$3`).
			WithArgs(int64(0), credentialDigestVersionHMAC, 10).
			WillReturnRows(sqlmock.NewRows([]string{"id", "credentials", "credentials_digest_version", "credentials_digest", "credentials_api_key_digest"}).
				AddRow(int64(1), raw, nil, legacyDigest, *legacyAPIKeyDigest))
		mock.ExpectExec(`(?s)UPDATE accounts\s+SET credentials_digest = \$1,\s+credentials_api_key_digest = \$2,\s+credentials_digest_version = \$3\s+WHERE id = \$4\s+AND credentials = \$5::jsonb\s+AND credentials_digest_version IS NOT DISTINCT FROM \$6\s+AND credentials_digest IS NOT DISTINCT FROM \$7\s+AND credentials_api_key_digest IS NOT DISTINCT FROM \$8`).
			WithArgs(hmacDigest, *hmacAPIKeyDigest, credentialDigestVersionHMAC, int64(1), string(raw), nil, legacyDigest, *legacyAPIKeyDigest).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(`(?s)SELECT id, credentials, credentials_digest_version, credentials_digest, credentials_api_key_digest.*WHERE id > \$1.*ORDER BY id.*LIMIT \$3`).
			WithArgs(int64(1), credentialDigestVersionHMAC, 10).
			WillReturnRows(sqlmock.NewRows([]string{"id", "credentials", "credentials_digest_version", "credentials_digest", "credentials_api_key_digest"}))

		updated, skipped, err := BackfillAccountCredentialDigests(context.Background(), db, &credentialCipher{hmacKey: []byte("stable-test-hmac-key")}, 10)
		require.NoError(t, err)
		require.EqualValues(t, 1, updated)
		require.Zero(t, skipped)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("undecryptable rows are left unchanged and reported as skipped", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })
		badCiphertext := []byte(`{"api_key":"enc:v1:unreadable"}`)
		mock.ExpectQuery(`(?s)SELECT id, credentials, credentials_digest_version, credentials_digest, credentials_api_key_digest.*WHERE id > \$1.*ORDER BY id.*LIMIT \$3`).
			WithArgs(int64(0), credentialDigestVersionHMAC, 10).
			WillReturnRows(sqlmock.NewRows([]string{"id", "credentials", "credentials_digest_version", "credentials_digest", "credentials_api_key_digest"}).
				AddRow(int64(9), badCiphertext, nil, "old-digest", nil))
		mock.ExpectQuery(`(?s)SELECT id, credentials, credentials_digest_version, credentials_digest, credentials_api_key_digest.*WHERE id > \$1.*ORDER BY id.*LIMIT \$3`).
			WithArgs(int64(9), credentialDigestVersionHMAC, 10).
			WillReturnRows(sqlmock.NewRows([]string{"id", "credentials", "credentials_digest_version", "credentials_digest", "credentials_api_key_digest"}))

		updated, skipped, err := BackfillAccountCredentialDigests(context.Background(), db, &credentialCipher{hmacKey: []byte("stable-test-hmac-key")}, 10)
		require.NoError(t, err)
		require.Zero(t, updated)
		require.EqualValues(t, 1, skipped)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCredentialCipherStrictRestoreRejectsUnreadableEnvelopes(t *testing.T) {
	cipher := testCipher(t)
	stored := mustPrepareCredentials(t, cipher, map[string]any{"api_key": "secret"})
	restored, err := cipher.restoreForReadStrict(stored)
	require.NoError(t, err)
	require.Equal(t, "secret", restored["api_key"])

	_, err = (&credentialCipher{}).restoreForReadStrict(stored)
	require.Error(t, err, "ciphertext must not be mistaken for a plaintext credential when the key is unavailable")

	wrongKey := &credentialCipher{encryptor: failingCredentialEncryptor{err: errors.New("bad key")}}
	_, err = wrongKey.restoreForReadStrict(stored)
	require.Error(t, err)
}

func TestBackfillAccountCredentialDigestsDecryptsBeforeHashingAndPreservesConcurrentWrites(t *testing.T) {
	for _, affected := range []int64{0, 1} {
		t.Run(map[int64]string{0: "concurrent update wins", 1: "unchanged row migrates"}[affected], func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })

			cipher := testCipher(t)
			cipher.hmacKey = []byte("stable-test-hmac-key")
			plain := map[string]any{"api_key": "legacy-key", "refresh_token": "legacy-refresh"}
			stored := mustPrepareCredentials(t, cipher, plain)
			raw, err := json.Marshal(stored)
			require.NoError(t, err)
			_, digest, legacyDigest, apiDigest, legacyAPIDigest, err := credentialDigestCandidates(plain, cipher.hmacKey)
			require.NoError(t, err)

			query := `(?s)SELECT id, credentials, credentials_digest_version, credentials_digest, credentials_api_key_digest.*WHERE id > \$1.*ORDER BY id.*LIMIT \$3`
			mock.ExpectQuery(query).WithArgs(int64(0), credentialDigestVersionHMAC, 10).
				WillReturnRows(sqlmock.NewRows([]string{"id", "credentials", "credentials_digest_version", "credentials_digest", "credentials_api_key_digest"}).
					AddRow(int64(7), raw, credentialDigestVersionLegacy, legacyDigest, *legacyAPIDigest))
			mock.ExpectExec(`(?s)UPDATE accounts.*AND credentials = \$5::jsonb.*AND credentials_digest_version IS NOT DISTINCT FROM \$6.*AND credentials_digest IS NOT DISTINCT FROM \$7.*AND credentials_api_key_digest IS NOT DISTINCT FROM \$8`).
				WithArgs(digest, *apiDigest, credentialDigestVersionHMAC, int64(7), string(raw), credentialDigestVersionLegacy, legacyDigest, *legacyAPIDigest).
				WillReturnResult(sqlmock.NewResult(0, affected))
			mock.ExpectQuery(query).WithArgs(int64(7), credentialDigestVersionHMAC, 10).
				WillReturnRows(sqlmock.NewRows([]string{"id", "credentials", "credentials_digest_version", "credentials_digest", "credentials_api_key_digest"}))

			updated, skipped, err := BackfillAccountCredentialDigests(context.Background(), db, cipher, 10)
			require.NoError(t, err)
			require.Equal(t, affected, updated)
			require.Zero(t, skipped)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestBackfillAccountCredentialDigestsRepairsChangedKeysWithoutRewritingCurrentRows(t *testing.T) {
	for _, current := range []bool{false, true} {
		t.Run(map[bool]string{false: "changed key", true: "current key"}[current], func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })
			cipher := &credentialCipher{hmacKey: []byte("current-hmac-key")}
			credentials := map[string]any{"refresh_token": "refresh"}
			_, digest, _, _, _, err := credentialDigestCandidates(credentials, cipher.hmacKey)
			require.NoError(t, err)
			storedDigest := "previous-key-digest"
			if current {
				storedDigest = digest
			}
			query := `(?s)SELECT id, credentials, credentials_digest_version, credentials_digest, credentials_api_key_digest.*credentials_digest_version IS NULL OR credentials_digest_version IN \(1, \$2\).*ORDER BY id.*LIMIT \$3`
			mock.ExpectQuery(query).WithArgs(int64(0), credentialDigestVersionHMAC, 10).
				WillReturnRows(sqlmock.NewRows([]string{"id", "credentials", "credentials_digest_version", "credentials_digest", "credentials_api_key_digest"}).
					AddRow(int64(1), `{"refresh_token":"refresh"}`, credentialDigestVersionHMAC, storedDigest, nil))
			if !current {
				mock.ExpectExec(`(?s)UPDATE accounts.*credentials_digest_version IS NOT DISTINCT FROM`).
					WithArgs(digest, nil, credentialDigestVersionHMAC, int64(1), `{"refresh_token":"refresh"}`, credentialDigestVersionHMAC, storedDigest, nil).
					WillReturnResult(sqlmock.NewResult(0, 1))
			}
			mock.ExpectQuery(query).WithArgs(int64(1), credentialDigestVersionHMAC, 10).
				WillReturnRows(sqlmock.NewRows([]string{"id", "credentials", "credentials_digest_version", "credentials_digest", "credentials_api_key_digest"}))
			updated, skipped, err := BackfillAccountCredentialDigests(context.Background(), db, cipher, 10)
			require.NoError(t, err)
			require.Zero(t, skipped)
			if current {
				require.Zero(t, updated)
			} else {
				require.EqualValues(t, 1, updated)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// 敏感键清单：加密与脱敏必须覆盖同一组键。
func TestSensitiveKeysCoveredByEncryption(t *testing.T) {
	for _, key := range service.SensitiveCredentialKeys {
		if !service.IsSensitiveCredentialKey(key) {
			t.Fatalf("key %q in list but not detected as sensitive", key)
		}
	}
}

// probeTestDigest 从凭据 JSON 字符串计算文档摘要，供 SQL mock 断言使用。
func probeTestDigest(t *testing.T, credsJSON string) string {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(credsJSON), &m))
	_, d, _, err := canonicalCredentialsPayload(m)
	require.NoError(t, err)
	return d
}

// probeTestAPIKeyDigest 从凭据 JSON 文档计算 api_key 值摘要，供 SQL mock 断言使用。
func probeTestAPIKeyDigest(t *testing.T, credsJSON string) string {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(credsJSON), &m))
	_, _, ak, err := canonicalCredentialsPayload(m)
	require.NoError(t, err)
	if ak == nil {
		t.Fatal("credentials document has no api_key")
	}
	return *ak
}

// 测试辅助：十六进制密钥合法性（与 AESEncryptor 约定一致）。
func TestTestKeyHexLength(t *testing.T) {
	raw, err := hex.DecodeString(testEncryptionKeyHex)
	if err != nil || len(raw) != 32 {
		t.Fatalf("test key must decode to 32 bytes, got %d bytes err=%v", len(raw), err)
	}
}
