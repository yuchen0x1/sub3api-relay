package repository

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

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
	if c := NewAccountCredentialCipher(nil); c.encryptor != nil {
		t.Fatal("nil config must yield passthrough cipher")
	}
	cfg := &config.Config{}
	cfg.Totp.EncryptionKey = testEncryptionKeyHex
	cfg.Totp.EncryptionKeyConfigured = false
	if c := NewAccountCredentialCipher(cfg); c.encryptor != nil {
		t.Fatal("unconfigured key must yield passthrough cipher")
	}
	cfg.Totp.EncryptionKeyConfigured = true
	if c := NewAccountCredentialCipher(cfg); c.encryptor == nil {
		t.Fatal("configured key must enable encryption")
	}
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
