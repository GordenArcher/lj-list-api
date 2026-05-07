ALTER TABLE applications
    DROP CONSTRAINT IF EXISTS applications_package_type_check;

ALTER TABLE applications
    ADD CONSTRAINT applications_package_type_check
    CHECK (package_type IN ('fixed', 'provisions', 'detergents', 'custom'));
