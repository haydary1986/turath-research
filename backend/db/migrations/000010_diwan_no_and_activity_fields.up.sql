-- ══════════════════════════════════════════════════
-- Migration 10: official registry number on letters + richer activity fields
-- ══════════════════════════════════════════════════

-- Letters are legally issued under the presidency diwan's outgoing-mail number
-- (رقم صادرة الديوان); the internal letter_no stays for online verification.
ALTER TABLE issued_letters ADD COLUMN diwan_no VARCHAR(100);

-- Ministry activity reports ask for the audience, the organizing body and
-- whether the activity was in person or online.
ALTER TABLE activities ADD COLUMN target_audience VARCHAR(255);
ALTER TABLE activities ADD COLUMN organizer       VARCHAR(255);
ALTER TABLE activities ADD COLUMN attendance_mode VARCHAR(20)
    CHECK (attendance_mode IN ('in_person', 'online', 'hybrid'));
