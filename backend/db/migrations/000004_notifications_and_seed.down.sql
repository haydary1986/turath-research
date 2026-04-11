DROP TABLE IF EXISTS notifications CASCADE;

-- Remove seed data
DELETE FROM publication_authors WHERE publication_id IN (SELECT id FROM publications WHERE id::text LIKE 'e000%');
DELETE FROM publications WHERE id::text LIKE 'e000%';
DELETE FROM journals WHERE id::text LIKE 'f000%';
DELETE FROM researchers WHERE id::text LIKE 'a000%';
DELETE FROM departments WHERE id::text LIKE 'd000%';
DELETE FROM colleges WHERE id::text LIKE 'b000%';
DELETE FROM institutions WHERE id = '11111111-1111-1111-1111-111111111111';
DELETE FROM users WHERE id::text LIKE '10000%';
DELETE FROM incentive_rules WHERE institution_id = '11111111-1111-1111-1111-111111111111';
DELETE FROM promotion_requirements WHERE institution_id = '11111111-1111-1111-1111-111111111111';
