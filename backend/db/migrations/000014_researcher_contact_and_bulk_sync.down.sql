DROP TABLE IF EXISTS bulk_sync_runs;

ALTER TABLE researchers
    DROP COLUMN IF EXISTS phone,
    DROP COLUMN IF EXISTS phone_secondary,
    DROP COLUMN IF EXISTS recovery_email,
    DROP COLUMN IF EXISTS web_of_science_url,
    DROP COLUMN IF EXISTS employee_no,
    DROP COLUMN IF EXISTS account_status;
