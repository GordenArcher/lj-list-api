ALTER TABLE conversations
    DROP COLUMN IF EXISTS last_customer_message_at,
    DROP COLUMN IF EXISTS pending_customer_message_count,
    DROP COLUMN IF EXISTS last_admin_message_sms_at;
