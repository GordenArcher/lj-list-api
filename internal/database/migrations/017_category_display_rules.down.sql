ALTER TABLE categories
    DROP COLUMN IF EXISTS orderable,
    DROP COLUMN IF EXISTS requires_inquiry,
    DROP COLUMN IF EXISTS tag,
    DROP COLUMN IF EXISTS instructions,
    DROP COLUMN IF EXISTS description;
