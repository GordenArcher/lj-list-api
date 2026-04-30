CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    password_hash VARCHAR(255) NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    phone_number VARCHAR(20) NOT NULL,
    staff_number VARCHAR(100) NOT NULL,
    institution VARCHAR(255) NOT NULL,
    ghana_card_number VARCHAR(50) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    otp_hash VARCHAR(255),
    otp_expires_at TIMESTAMPTZ,
    role VARCHAR(20) NOT NULL DEFAULT 'customer' CHECK (role IN ('customer', 'admin')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_users_phone_number ON users (phone_number);
CREATE UNIQUE INDEX idx_users_staff_number ON users (staff_number);
CREATE UNIQUE INDEX idx_users_ghana_card_number ON users (ghana_card_number);
CREATE INDEX idx_users_role ON users (role);
