-- Per-account traffic weight used by the scheduler's weighted selection mode.
-- Values <= 0 are treated as the default weight of 1 at selection time.
ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS weight INT NOT NULL DEFAULT 1;
