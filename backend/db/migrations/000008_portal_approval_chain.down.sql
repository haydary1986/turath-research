UPDATE portal_requests SET status = 'pending' WHERE status IN ('pending_dean', 'pending_vp');
ALTER TABLE portal_requests DROP COLUMN IF EXISTS sa_decision;
ALTER TABLE portal_requests DROP CONSTRAINT portal_requests_status_check;
ALTER TABLE portal_requests ADD CONSTRAINT portal_requests_status_check
    CHECK (status IN ('draft', 'pending', 'returned', 'approved', 'rejected', 'verified',
                      'not_eligible', 'awaiting_signatures', 'completed', 'issued'));
