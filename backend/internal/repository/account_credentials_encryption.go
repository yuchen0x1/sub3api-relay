package repository

import (
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

// credentialCipher 对 Account.Credentials 的敏感子键做字段级加解密。
// encryptor 为 nil 时（TOTP_ENCRYPTION_KEY 未显式配置）保持明文直通——
// 随机临时密钥跨进程不可解密，绝不能用于持久化凭据。
type credentialCipher struct {
	encryptor service.SecretEncryptor
}

// NewAccountCredentialCipher 构造凭据字段级加密器。
// 仅当加密密钥被显式配置(EncryptionKeyConfigured)时启用加密；未配置时返回
// 明文直通 cipher，行为与历史版本一致。
func NewAccountCredentialCipher(cfg *config.Config) *credentialCipher {
	if cfg == nil || !cfg.Totp.EncryptionKeyConfigured {
		slog.Warn("account credential encryption disabled: TOTP_ENCRYPTION_KEY not explicitly configured; credentials stored as plaintext")
		return &credentialCipher{}
	}
	encryptor, err := NewAESEncryptor(cfg)
	if err != nil {
		// Load() 已校验密钥格式，此处失败属于程序性错误，直接拒绝启用加密
		// 而不是静默降级（配置了坏密钥比不加密更危险）。
		slog.Error("account credential encryption disabled: invalid encryption key", "err", err)
		return &credentialCipher{}
	}
	return &credentialCipher{encryptor: encryptor}
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

// prepareCredentialsStorage 返回可直接写入 DB 的凭据文档与摘要指针，
// 供 ent builder 链式调用使用。摘要永远基于明文 canonical payload 计算，
// 与加密开关无关（单向摘要不含明文信息，提前写入保证加密开关切换时
// CAS 守卫语义一致）。
func prepareCredentialsStorage(credCipher *credentialCipher, credentials map[string]any) (doc map[string]any, docDigest *string, apiKeyDigest *string, err error) {
	payload, digest, apiKeyDigest, err := canonicalCredentialsPayload(credentials)
	if err != nil {
		return nil, nil, nil, err
	}
	_ = payload
	doc = normalizeJSONMap(credentials)
	if credCipher != nil {
		doc = credCipher.prepareForWrite(doc)
	}
	return doc, &digest, apiKeyDigest, nil
}

// credentialsChangedClause 返回"凭据是否已变化"的守卫片段（用于 IS DISTINCT
// 语义）。旧行(digest 为 NULL)退回明文 jsonb 直比；新行走确定性摘要比较。
// payloadPH 绑定规范化明文 payload，digestPH 绑定其 SHA-256 十六进制。
// 注意：不能用 COALESCE——`NULL IS DISTINCT FROM x` 为 TRUE，会吞掉回退分支。
func credentialsChangedClause(credCol, digestCol, payloadPH, digestPH string) string {
	return "(CASE WHEN " + digestCol + " IS NULL THEN " + credCol + " IS DISTINCT FROM " + payloadPH + "::jsonb" +
		" ELSE " + digestCol + " IS DISTINCT FROM " + digestPH + " END)"
}

// credentialsEqualClause 返回"凭据是否未变化"的守卫片段（用于 = 语义的 CAS）。
func credentialsEqualClause(credCol, digestCol, payloadPH, digestPH string) string {
	return "(CASE WHEN " + digestCol + " IS NULL THEN " + credCol + " = " + payloadPH + "::jsonb" +
		" ELSE " + digestCol + " = " + digestPH + " END)"
}

// credentialsAPIKeyEqualClause 返回"api_key 值是否相同"的守卫片段，
// 供 Ollama 云用量组身份判定在 api_key 加密后继续工作。
func credentialsAPIKeyEqualClause(credCol, digestCol, payloadPH, digestPH string) string {
	return "(CASE WHEN " + digestCol + " IS NULL THEN " + credCol + " -> 'api_key' IS NOT DISTINCT FROM " + payloadPH + "::jsonb -> 'api_key'" +
		" ELSE " + digestCol + " = " + digestPH + " END)"
}

// credentialsDigestFromJSON 对任意来源的凭据 JSON 字符串规范化后计算摘要，
// 供 CAS 守卫绑定（快照里的 CredentialsJSON 未必是规范化字节序）。
// 解析失败返回 nil：守卫退回明文直比分支。
func credentialsDigestFromJSON(raw string) *string {
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
func (c *credentialCipher) prepareForWrite(credentials map[string]any) map[string]any {
	if len(credentials) == 0 || c.encryptor == nil {
		return credentials
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
			// 序列化失败的值保持原样写入：该值本就无法通过 jsonb 往返，
			// 由既有写入校验兜底，这里不放大错误面。
			out[key] = value
			continue
		}
		enc, err := c.encryptor.Encrypt(string(plain))
		if err != nil {
			slog.Warn("encrypt account credential value failed; storing plaintext", "key", key, "err", err)
			out[key] = value
			continue
		}
		// AESEncryptor.Encrypt 已输出 base64，信封直接拼接。
		out[key] = credentialEnvelopePrefix + enc
		changed = true
	}
	if !changed {
		return credentials
	}
	return out
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
