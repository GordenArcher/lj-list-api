ALTER TABLE applications
    ADD COLUMN package_id VARCHAR(255);

CREATE INDEX idx_applications_package_type_package_id
    ON applications (package_type, package_id);

CREATE INDEX idx_applications_fixed_package_name
    ON applications (package_name)
    WHERE package_type = 'fixed' AND package_id IS NULL;
