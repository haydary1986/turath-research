-- Reverse of 000003

DROP TABLE IF EXISTS approval_workflows CASCADE;
DROP TABLE IF EXISTS strategic_kpis CASCADE;
DROP TABLE IF EXISTS accreditation_recommendations CASCADE;
DROP TABLE IF EXISTS accreditation_evidence CASCADE;
DROP TABLE IF EXISTS research_center_members CASCADE;
DROP TABLE IF EXISTS research_centers CASCADE;
DROP TABLE IF EXISTS research_chairs CASCADE;
DROP TABLE IF EXISTS agreements CASCADE;
DROP TABLE IF EXISTS conference_participations CASCADE;
DROP TABLE IF EXISTS sabbatical_requests CASCADE;
DROP TABLE IF EXISTS annual_performance_reviews CASCADE;
DROP TABLE IF EXISTS performance_criteria_weights CASCADE;
DROP TABLE IF EXISTS publication_incentives CASCADE;
DROP TABLE IF EXISTS incentive_rules CASCADE;
DROP TABLE IF EXISTS promotion_applications CASCADE;
DROP TABLE IF EXISTS promotion_requirements CASCADE;

-- Remove added columns from users
ALTER TABLE users DROP COLUMN IF EXISTS last_login_at;
ALTER TABLE users DROP COLUMN IF EXISTS can_approve_budgets;
ALTER TABLE users DROP COLUMN IF EXISTS can_approve_promotions;
ALTER TABLE users DROP COLUMN IF EXISTS department_id;
ALTER TABLE users DROP COLUMN IF EXISTS college_id;

-- Revert user_role enum
ALTER TABLE users ALTER COLUMN role DROP DEFAULT;
ALTER TABLE users ALTER COLUMN role TYPE VARCHAR(50);
DROP TYPE IF EXISTS user_role;
CREATE TYPE user_role AS ENUM ('admin', 'institution_admin', 'college_admin', 'department_admin', 'researcher', 'viewer');
ALTER TABLE users ALTER COLUMN role TYPE user_role USING role::user_role;
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'viewer';

-- Drop new types
DROP TYPE IF EXISTS performance_criterion;
DROP TYPE IF EXISTS workflow_decision;
DROP TYPE IF EXISTS approver_role;
DROP TYPE IF EXISTS workflow_type;
DROP TYPE IF EXISTS kpi_scope;
DROP TYPE IF EXISTS kpi_source;
DROP TYPE IF EXISTS kpi_period;
DROP TYPE IF EXISTS kpi_category;
DROP TYPE IF EXISTS recommendation_status;
DROP TYPE IF EXISTS recommendation_type;
DROP TYPE IF EXISTS evidence_status;
DROP TYPE IF EXISTS evidence_type;
DROP TYPE IF EXISTS accreditation_body;
DROP TYPE IF EXISTS center_role;
DROP TYPE IF EXISTS center_status;
DROP TYPE IF EXISTS chair_status;
DROP TYPE IF EXISTS agreement_status;
DROP TYPE IF EXISTS agreement_type;
DROP TYPE IF EXISTS performance_status;
DROP TYPE IF EXISTS incentive_status;
DROP TYPE IF EXISTS journal_classification;
DROP TYPE IF EXISTS participation_type;
DROP TYPE IF EXISTS conference_status;
DROP TYPE IF EXISTS sabbatical_status;
DROP TYPE IF EXISTS promotion_status;
DROP TYPE IF EXISTS field_category;
DROP TYPE IF EXISTS academic_rank;
