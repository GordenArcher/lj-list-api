DROP INDEX IF EXISTS idx_applications_fixed_package_name;
DROP INDEX IF EXISTS idx_applications_package_type_package_id;

ALTER TABLE applications
    DROP COLUMN IF EXISTS package_id;
