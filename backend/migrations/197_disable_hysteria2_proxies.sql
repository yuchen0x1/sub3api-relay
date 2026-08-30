-- Hysteria2's reachable client dependency has an unfixed OOM advisory.
-- Retain rows for operator migration, but fail closed by deactivating them.
UPDATE proxies
SET status = 'inactive', updated_at = NOW()
WHERE protocol = 'hysteria2'
  AND status = 'active';
