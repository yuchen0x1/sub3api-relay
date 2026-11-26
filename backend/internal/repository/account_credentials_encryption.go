package repository

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/yuchen0x1/sub3api-relay/internal/config"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

// credentialEnvelopePrefix 标记已加密的凭据值信封。凭据 JSONB 中敏感子键的值
// 以 "enc:v1:" + base64(AES-256-GCM(json(明文值))) 存储，JSON 键结构保持不变，
// 以便 `credentials ? 'refresh_token'`、`credentials->>'base_url'` 等 SQL 侧
// JSON-path 查询继续工作；语义相等比较(CAS/去重)改走 credentials_digest。
const credentialEnvelopePrefix = "enc:v1:"

const (
	credentialDigestVersionHMAC   int16 = 2
	credentialDigestVersionLegacy int16 = 1
	credentialDigestHMACContext         = "sub3api-credentials-digest-v2"
	credentialDigestDomain              = "account.credentials"
	apiKeyDigestDomain                  = "account.credentials.api_key"
	credentialDigestKeyContext          = "sub3api/account-credentials/digest-key/v2"
)

// credentialCipher 对 Account.Credentials 的敏感子键做字段级加解密。
// encryptor 为 nil 时（TOTP_ENCRYPTION_KEY 未显式配置）保持明文直通；
// hmacKey 优先派生自外部 TOTP 加密密钥，未启用加密时才使用持久化 JWT secret。
type credentialCipher struct {
	encryptor service.SecretEncryptor
	initErr   error
	// hmacKey is domain-derived independently from the encryption operation and
	// is never exposed to SQL or logs.
	hmacKey []byte
}

// NewAccountCredentialCipher 构造凭据字段级加密器与稳定的凭据摘要密钥。
// 凭据加密仅在 TOTP 加密密钥显式配置时启用；摘要密钥从启动时已持久化的
// TOTP 加密密钥派生，避免数据库中的 JWT secret 削弱密文的离线猜测防护。
// 未启用凭据加密时，使用已持久化的 JWT secret 保持摘要稳定。
func NewAccountCredentialCipher(cfg *config.Config) *credentialCipher {
	if cfg == nil {
		return &credentialCipher{initErr: fmt.Errorf("account credential cipher requires config with a stable digest key")}
	}
	hmacKey := deriveCredentialDigestKey(cfg.JWT.Secret)
	if !cfg.Totp.EncryptionKeyConfigured {
		if len(hmacKey) == 0 {
			err := fmt.Errorf("JWT secret must be available before account credential digest initialization")
			slog.Error("account credential writes blocked: credential digest key unavailable", "err", err)
			return &credentialCipher{initErr: err}
		}
		slog.Warn("account credential encryption disabled: TOTP_ENCRYPTION_KEY not explicitly configured; credentials stored as plaintext")
		return &credentialCipher{hmacKey: hmacKey}
	}
	encryptor, err := NewAESEncryptor(cfg)
	if err != nil {
		// Load() 通常已校验密钥格式。即使绕过该校验，后续凭据写入也必须
		// 显式失败，不能把配置错误当成主动关闭加密并写入明文。
		slog.Error("account credential writes blocked: invalid encryption key", "err", err)
		return &credentialCipher{initErr: err}
	}
	key, keyErr := hex.DecodeString(strings.TrimSpace(cfg.Totp.EncryptionKey))
	if keyErr != nil || len(key) != 32 {
		if keyErr == nil {
			keyErr = fmt.Errorf("totp encryption key must be 32 bytes")
		}
		return &credentialCipher{initErr: keyErr}
	}
	hmacKey = deriveCredentialDigestKey(string(key))
	return &credentialCipher{encryptor: encryptor, hmacKey: append([]byte(nil), hmacKey...)}
}

func deriveCredentialDigestKey(master string) []byte {
	if master == "" {
		return nil
	}
	h := hmac.New(sha256.New, []byte(master))
	_, _ = h.Write([]byte(credentialDigestKeyContext))
	return h.Sum(nil)
}

// canonicalCredentialsPayload 生成凭据文档的规范化明文 payload 及其摘要。
// 摘要永远基于明文计算，供 CAS/去重守卫在字段级加密后做语义相等比较；
// payload 同时作为遗留明文行的直比 bind。apiKeyDigest 为 credentials.api_key
// 规范化值的摘要，键不存在时为 nil。
func canonicalCredentialsPayload(credentials map[string]any) (payload []byte, digest string, apiKeyDigest *string, err error) {
	payload, err = json.Marshal(normalizeJSONMap(credentials))
	if err != nil {
		return nil, "", nil, fmt.Errorf("marshal credentials: %w", err)
	}
	sum := sha256.Sum256(payload)
	digest = hex.EncodeToString(sum[:])
	if _, ok := credentials["api_key"]; ok {
		if raw, mErr := json.Marshal(credentials["api_key"]); mErr == nil {
			apiKeySum := sha256.Sum256(raw)
			hexDigest := hex.EncodeToString(apiKeySum[:])
			apiKeyDigest = &hexDigest
		}
	}
	return payload, digest, apiKeyDigest, nil
}

// secureCredentialDigests computes versioned, domain-separated HMAC-SHA-256
// digests. The legacy SHA-256 values remain available for dual-read migration.
func secureCredentialDigests(payload []byte, apiKeyJSON []byte, key []byte) (string, *string) {
	if len(key) == 0 {
		return "", nil
	}
	digest := credentialHMAC(key, credentialDigestDomain, payload)
	var apiKeyDigest *string
	if apiKeyJSON != nil {
		value := credentialHMAC(key, apiKeyDigestDomain, apiKeyJSON)
		apiKeyDigest = &value
	}
	return digest, apiKeyDigest
}

func credentialHMAC(key []byte, domain string, payload []byte) string {
	h := hmac.New(sha256.New, key)
	_, _ = h.Write([]byte(credentialDigestHMACContext))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(domain))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

func credentialDigestVersionForKey(key []byte) int16 {
	if len(key) == 0 {
		return credentialDigestVersionLegacy
	}
	return credentialDigestVersionHMAC
}

// credentialDigestCandidates returns the new HMAC digest first and the legacy
// SHA-256 digest second. The latter is used only for reading/CAS matching old
// rows; all fixed-key writes use the HMAC value.
func credentialDigestCandidates(credentials map[string]any, key []byte) (payload []byte, primary string, legacy string, apiPrimary *string, apiLegacy *string, err error) {
	payload, legacy, apiLegacy, err = canonicalCredentialsPayload(credentials)
	if err != nil {
		return nil, "", "", nil, nil, err
	}
	var apiKeyJSON []byte
	if _, ok := credentials["api_key"]; ok {
		apiKeyJSON, err = json.Marshal(credentials["api_key"])
		if err != nil {
			return nil, "", "", nil, nil, fmt.Errorf("marshal api_key: %w", err)
		}
	}
	primary, apiPrimary = secureCredentialDigests(payload, apiKeyJSON, key)
	if primary == "" {
		primary = legacy
		apiPrimary = apiLegacy
	}
	return payload, primary, legacy, apiPrimary, apiLegacy, nil
}

// prepareCredentialsStorage 返回可直接写入 DB 的凭据文档与摘要指针，
// 供 ent builder 链式调用使用。摘要永远基于明文 canonical payload 计算，
// 使用稳定密钥时写入 HMAC；无密钥仅供包内遗留格式测试。
// 公开仓储构造函数拒绝缺失 cipher，生产写入不会降级为 SHA-256。
func prepareCredentialsStorage(credCipher *credentialCipher, credentials map[string]any) (doc map[string]any, docDigest *string, apiKeyDigest *string, err error) {
	_, digest, _, apiKeyDigest, _, err := credentialDigestCandidates(credentials, digestKey(credCipher))
	if err != nil {
		return nil, nil, nil, err
	}
	doc = normalizeJSONMap(credentials)
	if credCipher != nil {
		doc, err = credCipher.prepareForWrite(doc)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	return doc, &digest, apiKeyDigest, nil
}

func digestKey(cipher *credentialCipher) []byte {
	if cipher == nil {
		return nil
	}
	return cipher.hmacKey
}

// credentialsChangedClause 返回"凭据是否已变化"的守卫片段（用于 IS DISTINCT
// 语义）。旧行(digest 为 NULL)退回明文 jsonb 直比；新行走确定性摘要比较。
// payloadPH 绑定规范化明文 payload，摘要参数分别绑定 HMAC 与遗留 SHA-256。
// 注意：不能用 COALESCE——`NULL IS DISTINCT FROM x` 为 TRUE，会吞掉回退分支。
func credentialsChangedClause(credCol, digestCol, versionCol, payloadPH, digestPH, legacyDigestPH string) string {
	return "(CASE WHEN " + versionCol + " = 2 THEN " + digestCol + " IS DISTINCT FROM " + digestPH +
		" WHEN " + versionCol + " = 1 OR " + versionCol + " IS NULL THEN CASE WHEN " + digestCol + " IS NULL THEN " + credCol + " IS DISTINCT FROM " + payloadPH + "::jsonb" +
		" ELSE " + digestCol + " IS DISTINCT FROM " + legacyDigestPH + " END ELSE TRUE END)"
}

// credentialsEqualClause 返回"凭据是否未变化"的守卫片段（用于 = 语义的 CAS）。
func credentialsEqualClause(credCol, digestCol, versionCol, payloadPH, digestPH, legacyDigestPH string) string {
	return "(CASE WHEN " + versionCol + " = 2 THEN " + digestCol + " = " + digestPH +
		" WHEN " + versionCol + " = 1 OR " + versionCol + " IS NULL THEN CASE WHEN " + digestCol + " IS NULL THEN " + credCol + " = " + payloadPH + "::jsonb" +
		" ELSE " + digestCol + " = " + legacyDigestPH + " END ELSE FALSE END)"
}

// credentialsAPIKeyEqualClause 返回"api_key 值是否相同"的守卫片段，
// 供 Ollama 云用量组身份判定在 api_key 加密后继续工作。
func credentialsAPIKeyEqualClause(credCol, digestCol, versionCol, payloadPH, digestPH, legacyDigestPH string) string {
	return "(CASE WHEN " + versionCol + " = 2 THEN " + digestCol + " IS NOT DISTINCT FROM " + digestPH +
		" WHEN " + versionCol + " = 1 OR " + versionCol + " IS NULL THEN CASE WHEN " + digestCol + " IS NULL THEN " + credCol + " -> 'api_key' IS NOT DISTINCT FROM " + payloadPH + "::jsonb -> 'api_key'" +
		" ELSE " + digestCol + " = " + legacyDigestPH + " END ELSE FALSE END)"
}

// credentialsDigestFromJSON 对任意来源的凭据 JSON 字符串规范化后计算摘要，
// 供 CAS 守卫绑定（快照里的 CredentialsJSON 未必是规范化字节序）。
// 解析失败返回 nil：守卫退回明文直比分支。
func credentialsDigestFromJSON(raw string, hmacKeys ...[]byte) *string {
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil
	}
	key := []byte(nil)
	if len(hmacKeys) > 0 {
		key = hmacKeys[0]
	}
	_, digest, _, _, _, err := credentialDigestCandidates(m, key)
	if err != nil {
		return nil
	}
	return &digest
}

func credentialDigestPairFromJSON(raw string, key []byte) (primary, legacy *string) {
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, nil
	}
	_, current, old, _, _, err := credentialDigestCandidates(m, key)
	if err != nil {
		return nil, nil
	}
	return &current, &old
}

func credentialDigestPair(credentials map[string]any, key []byte) (primary, legacy string, err error) {
	_, primary, legacy, _, _, err = credentialDigestCandidates(credentials, key)
	return primary, legacy, err
}

func apiKeyDigestPair(credentials map[string]any, key []byte) (primary, legacy *string, err error) {
	_, _, _, primary, legacy, err = credentialDigestCandidates(credentials, key)
	return primary, legacy, err
}

func legacyCredentialDigestFromJSON(raw string) *string {
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil
	}
	_, digest, _, err := canonicalCredentialsPayload(m)
	if err != nil {
		return nil
	}
	return &digest
}

// prepareForWrite 返回可直接写入 DB 的凭据副本：敏感子键值替换为 enc:v1: 信封。
// 非敏感子键(model_mapping、base_url 等)与 JSON 结构保持不变。
// 已带信封的值跳过（幂等），避免双重加密。
func (c *credentialCipher) prepareForWrite(credentials map[string]any) (map[string]any, error) {
	if c.initErr != nil {
		return nil, fmt.Errorf("account credential encryption unavailable: %w", c.initErr)
	}
	if len(credentials) == 0 || c.encryptor == nil {
		return credentials, nil
	}
	out := make(map[string]any, len(credentials))
	changed := false
	for key, value := range credentials {
		if !service.IsSensitiveCredentialKey(key) || value == nil || value == "" {
			out[key] = value
			continue
		}
		if s, isStr := value.(string); isStr && strings.HasPrefix(s, credentialEnvelopePrefix) {
			out[key] = value
			continue
		}
		plain, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("marshal account credential %q: %w", key, err)
		}
		enc, err := c.encryptor.Encrypt(string(plain))
		if err != nil {
			return nil, fmt.Errorf("encrypt account credential %q: %w", key, err)
		}
		// AESEncryptor.Encrypt 已输出 base64，信封直接拼接。
		out[key] = credentialEnvelopePrefix + enc
		changed = true
	}
	if !changed {
		return credentials, nil
	}
	return out, nil
}

// restoreForRead 返回解密后的凭据副本。解密失败的值保留信封原文并告警——
// 上游调用会以显式 401 暴露问题，绝不静默注入垃圾值。
func (c *credentialCipher) restoreForRead(credentials map[string]any) map[string]any {
	if len(credentials) == 0 || c.encryptor == nil {
		return credentials
	}
	out := make(map[string]any, len(credentials))
	for key, value := range credentials {
		out[key] = c.restoreValue(key, value)
	}
	return out
}

// restoreForReadStrict is used by credential migrations and CAS backfills.
// Unlike serving reads, it never returns an undecryptable envelope as if it
// were plaintext; callers must leave that row untouched and surface the error.
func (c *credentialCipher) restoreForReadStrict(credentials map[string]any) (map[string]any, error) {
	if len(credentials) == 0 {
		return credentials, nil
	}
	out := make(map[string]any, len(credentials))
	for key, value := range credentials {
		s, isString := value.(string)
		if !isString || !strings.HasPrefix(s, credentialEnvelopePrefix) {
			out[key] = value
			continue
		}
		if c == nil || c.encryptor == nil {
			return nil, fmt.Errorf("cannot decrypt account credential %q: encryption key is unavailable", key)
		}
		plain, err := c.encryptor.Decrypt(strings.TrimPrefix(s, credentialEnvelopePrefix))
		if err != nil {
			return nil, fmt.Errorf("decrypt account credential %q: %w", key, err)
		}
		var restored any
		if err := json.Unmarshal([]byte(plain), &restored); err != nil {
			return nil, fmt.Errorf("decode account credential %q: %w", key, err)
		}
		out[key] = restored
	}
	return out, nil
}

func (c *credentialCipher) restoreValue(key string, value any) any {
	s, isStr := value.(string)
	if !isStr || !strings.HasPrefix(s, credentialEnvelopePrefix) {
		return value
	}
	enc := strings.TrimPrefix(s, credentialEnvelopePrefix)
	plain, err := c.encryptor.Decrypt(enc)
	if err != nil {
		slog.Warn("decrypt account credential value failed; keeping ciphertext (check TOTP_ENCRYPTION_KEY)", "key", key, "err", err)
		return value
	}
	var restored any
	if err := json.Unmarshal([]byte(plain), &restored); err != nil {
		slog.Warn("unmarshal decrypted account credential failed; keeping ciphertext", "key", key, "err", err)
		return value
	}
	return restored
}
