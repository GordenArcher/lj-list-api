ALTER TABLE conversations
    ADD COLUMN last_admin_message_sms_at TIMESTAMPTZ,
    ADD COLUMN pending_customer_message_count INT NOT NULL DEFAULT 0,
    ADD COLUMN last_customer_message_at TIMESTAMPTZ;
