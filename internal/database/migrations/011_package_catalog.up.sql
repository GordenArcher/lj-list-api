CREATE TABLE fixed_packages (
    id TEXT PRIMARY KEY,
    sort_order INT NOT NULL DEFAULT 0,
    name VARCHAR(255) NOT NULL,
    tagline TEXT NOT NULL DEFAULT '',
    price VARCHAR(50) NOT NULL,
    monthly VARCHAR(50) NOT NULL,
    tag VARCHAR(100) NOT NULL DEFAULT '',
    popular BOOLEAN NOT NULL DEFAULT false,
    rice_options TEXT NOT NULL DEFAULT '',
    items JSONB NOT NULL DEFAULT '[]',
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE department_packages (
    id TEXT PRIMARY KEY,
    kind VARCHAR(20) NOT NULL CHECK (kind IN ('provisions', 'detergents')),
    sort_order INT NOT NULL DEFAULT 0,
    name VARCHAR(255) NOT NULL,
    price INT NOT NULL CHECK (price > 0),
    items TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fixed_packages_active ON fixed_packages (active);
CREATE INDEX idx_fixed_packages_sort_order ON fixed_packages (sort_order, name);
CREATE INDEX idx_department_packages_kind_active ON department_packages (kind, active);
CREATE INDEX idx_department_packages_sort_order ON department_packages (kind, sort_order, name);
