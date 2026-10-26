-- Four-stage approval chain: college dean (initial) -> scientific affairs
-- (decision raised as a recommendation) -> vice president (confirmation).
ALTER TABLE portal_requests DROP CONSTRAINT portal_requests_status_check;
ALTER TABLE portal_requests ADD CONSTRAINT portal_requests_status_check
    CHECK (status IN ('draft', 'pending_dean', 'pending', 'pending_vp', 'returned',
                      'approved', 'rejected', 'verified', 'not_eligible',
                      'awaiting_signatures', 'completed', 'issued'));

-- The outcome scientific affairs recommended, awaiting VP confirmation.
ALTER TABLE portal_requests ADD COLUMN sa_decision VARCHAR(30);

-- Requests already sitting with scientific affairs go back one step so the
-- dean's initial approval is never skipped.
UPDATE portal_requests SET status = 'pending_dean' WHERE status = 'pending';
