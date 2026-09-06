-- 账号凭据字段级加密的配套摘要列。
--
-- accounts.credentials 的敏感子键（access_token / refresh_token / api_key 等）在
-- TOTP_ENCRYPTION_KEY 显式配置时以 "enc:v1:" 信封做 AES-256-GCM 字段级加密。
-- 加密后 SQL 侧无法再用 jsonb 相等做 CAS / 去重 / 身份判定，改走以下确定性摘要：
--   - credentials_digest: 规范化明文凭据文档(json.Marshal)的 SHA-256 十六进制。
--     NULL = 加密启用前写入的旧行（仍为明文），守卫自动退回明文直比，
--     该行下次全量凭据写入时补齐摘要并转为加密存储。
--   - credentials_api_key_digest: credentials.api_key 规范化值的 SHA-256 十六进制，
--     供 Ollama 云用量组身份判定使用。NULL 含义同上。
-- 摘要为单向哈希，不含明文信息；两列均可为空，无需回填。

ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS credentials_digest TEXT,
    ADD COLUMN IF NOT EXISTS credentials_api_key_digest TEXT;

COMMENT ON COLUMN accounts.credentials_digest IS
    'SHA-256 hex of canonical plaintext credentials; enables CAS equality under field-level encryption. NULL = legacy plaintext row';
COMMENT ON COLUMN accounts.credentials_api_key_digest IS
    'SHA-256 hex of canonical credentials.api_key value; NULL = legacy plaintext row or key absent';
