ALTER TABLE users
    ADD COLUMN IF NOT EXISTS phone_number VARCHAR(20),
    ADD COLUMN IF NOT EXISTS staff_number VARCHAR(100),
    ADD COLUMN IF NOT EXISTS institution VARCHAR(255),
    ADD COLUMN IF NOT EXISTS ghana_card_number VARCHAR(50),
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS otp_hash VARCHAR(255),
    ADD COLUMN IF NOT EXISTS otp_expires_at TIMESTAMPTZ;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'users'
          AND column_name = 'phone'
    ) THEN
        UPDATE users
        SET phone_number = CASE
            WHEN LEFT(BTRIM(phone), 1) = '+' THEN '+' || REGEXP_REPLACE(SUBSTRING(BTRIM(phone) FROM 2), '[^0-9]', '', 'g')
            ELSE REGEXP_REPLACE(BTRIM(phone), '[^0-9]', '', 'g')
        END
        WHERE phone_number IS NULL
          AND phone IS NOT NULL
          AND BTRIM(phone) <> '';
    END IF;
END $$;

WITH latest_application_fields AS (
    SELECT DISTINCT ON (user_id)
        user_id,
        staff_number,
        institution,
        ghana_card_number
    FROM applications
    ORDER BY user_id, created_at DESC
)
UPDATE users u
SET staff_number = COALESCE(u.staff_number, a.staff_number),
    institution = COALESCE(u.institution, a.institution),
    ghana_card_number = COALESCE(u.ghana_card_number, a.ghana_card_number)
FROM latest_application_fields a
WHERE u.id = a.user_id
  AND (u.staff_number IS NULL OR u.institution IS NULL OR u.ghana_card_number IS NULL);

UPDATE users
SET is_active = TRUE
WHERE otp_hash IS NULL
  AND otp_expires_at IS NULL;

ALTER TABLE users
    ALTER COLUMN is_active SET DEFAULT FALSE;

DROP INDEX IF EXISTS idx_users_email;

DROP INDEX IF EXISTS users_email_key;

ALTER TABLE users
    DROP COLUMN IF EXISTS email,
    DROP COLUMN IF EXISTS phone;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone_number ON users (phone_number);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_staff_number ON users (staff_number);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_ghana_card_number ON users (ghana_card_number);
