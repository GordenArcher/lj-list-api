CREATE TABLE applications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    package_type VARCHAR(20) NOT NULL CHECK (package_type IN ('fixed', 'custom')),
    package_name VARCHAR(255),
    cart_items JSONB DEFAULT '[]',
    total_amount INT NOT NULL CHECK (total_amount > 0),
    monthly_amount INT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'reviewed', 'approved', 'declined')),
    staff_number VARCHAR(100) NOT NULL,
    mandate_number VARCHAR(100) NOT NULL,
    institution VARCHAR(255) NOT NULL,
    ghana_card_number VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_applications_user_id ON applications (user_id);
CREATE INDEX idx_applications_status ON applications (status);
CREATE INDEX idx_applications_created_at ON applications (created_at DESC);
