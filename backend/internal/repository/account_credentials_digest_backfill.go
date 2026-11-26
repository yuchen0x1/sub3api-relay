package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
)

const defaultCredentialDigestBackfillBatchSize = 100

type credentialDigestBackfillRow struct {
	id           int64
	credentials  []byte
	version      sql.NullInt16
	digest       sql.NullString
	apiKeyDigest sql.NullString
}

// BackfillAccountCredentialDigests migrates legacy SHA digests to versioned
// HMAC digests in bounded, keyset-paginated batches. Every update compares the
// exact observed JSONB and digest columns so concurrent credential changes win.
// Rows that cannot be strictly decrypted are logged and left untouched.
func BackfillAccountCredentialDigests(ctx context.Context, db *sql.DB, cipher *credentialCipher, batchSize int) (updated, skipped int64, err error) {
	if db == nil {
		return 0, 0, errors.New("account credential digest backfill requires a database")
	}
	if cipher == nil || len(cipher.hmacKey) == 0 {
		return 0, 0, errors.New("account credential digest backfill requires a stable HMAC key")
	}
	if batchSize <= 0 {
		batchSize = defaultCredentialDigestBackfillBatchSize
	}
	if batchSize > 1000 {
		batchSize = 1000
	}

	var lastID int64
	for {
		rows, queryErr := db.QueryContext(ctx, `
			SELECT id, credentials, credentials_digest_version, credentials_digest, credentials_api_key_digest
			FROM accounts
			WHERE id > $1 AND (credentials_digest_version IS NULL OR credentials_digest_version IN (1, $2))
			ORDER BY id
			LIMIT $3
		`, lastID, credentialDigestVersionHMAC, batchSize)
		if queryErr != nil {
			return updated, skipped, fmt.Errorf("query account credential digest backfill batch: %w", queryErr)
		}

		batch := make([]credentialDigestBackfillRow, 0, batchSize)
		for rows.Next() {
			var row credentialDigestBackfillRow
			if scanErr := rows.Scan(&row.id, &row.credentials, &row.version, &row.digest, &row.apiKeyDigest); scanErr != nil {
				_ = rows.Close()
				return updated, skipped, fmt.Errorf("scan account credential digest backfill row: %w", scanErr)
			}
			batch = append(batch, row)
		}
		rowsErr := rows.Err()
		closeErr := rows.Close()
		if rowsErr != nil {
			return updated, skipped, fmt.Errorf("read account credential digest backfill batch: %w", rowsErr)
		}
		if closeErr != nil {
			return updated, skipped, fmt.Errorf("close account credential digest backfill rows: %w", closeErr)
		}
		if len(batch) == 0 {
			return updated, skipped, nil
		}

		for _, row := range batch {
			lastID = row.id
			var stored map[string]any
			if decodeErr := json.Unmarshal(row.credentials, &stored); decodeErr != nil {
				skipped++
				slog.Warn("account credential digest backfill skipped invalid credential JSON", "account_id", row.id, "err", decodeErr)
				continue
			}
			plain, decryptErr := cipher.restoreForReadStrict(stored)
			if decryptErr != nil {
				skipped++
				slog.Warn("account credential digest backfill skipped undecryptable credentials", "account_id", row.id, "err", decryptErr)
				continue
			}
			_, digest, _, apiKeyDigest, _, digestErr := credentialDigestCandidates(plain, cipher.hmacKey)
			if digestErr != nil {
				skipped++
				slog.Warn("account credential digest backfill skipped invalid credentials", "account_id", row.id, "err", digestErr)
				continue
			}
			if row.version.Valid && row.version.Int16 == credentialDigestVersionHMAC &&
				row.digest.Valid && row.digest.String == digest &&
				((apiKeyDigest == nil && !row.apiKeyDigest.Valid) ||
					(apiKeyDigest != nil && row.apiKeyDigest.Valid && row.apiKeyDigest.String == *apiKeyDigest)) {
				continue
			}

			result, execErr := db.ExecContext(ctx, `
				UPDATE accounts
				SET credentials_digest = $1,
					credentials_api_key_digest = $2,
					credentials_digest_version = $3
				WHERE id = $4
					AND credentials = $5::jsonb
					AND credentials_digest_version IS NOT DISTINCT FROM $6
					AND credentials_digest IS NOT DISTINCT FROM $7
					AND credentials_api_key_digest IS NOT DISTINCT FROM $8
			`, digest, apiKeyDigest, credentialDigestVersionHMAC, row.id, string(row.credentials),
				nullableInt16Arg(row.version), nullableStringArg(row.digest), nullableStringArg(row.apiKeyDigest))
			if execErr != nil {
				return updated, skipped, fmt.Errorf("update account credential digest for account %d: %w", row.id, execErr)
			}
			affected, affectedErr := result.RowsAffected()
			if affectedErr != nil {
				return updated, skipped, fmt.Errorf("read account credential digest update result for account %d: %w", row.id, affectedErr)
			}
			updated += affected
		}
	}
}

func nullableInt16Arg(value sql.NullInt16) any {
	if !value.Valid {
		return nil
	}
	return value.Int16
}

func nullableStringArg(value sql.NullString) any {
	if !value.Valid {
		return nil
	}
	return value.String
}
