-- The base users schema now includes the phone-number registration fields
-- and OTP activation columns. This migration only backfills older databases,
-- so rolling it back is a no-op.
SELECT 1;
