-- Records of the new categories keep their legacy type ('other'), so nothing is deleted.
DROP INDEX IF EXISTS idx_activities_category;
ALTER TABLE activities DROP CONSTRAINT IF EXISTS activities_infrastructure_count_check;
ALTER TABLE activities DROP CONSTRAINT IF EXISTS activities_category_check;
ALTER TABLE activities DROP COLUMN IF EXISTS notes;
ALTER TABLE activities DROP COLUMN IF EXISTS item_count;
ALTER TABLE activities DROP COLUMN IF EXISTS category_detail;
ALTER TABLE activities DROP COLUMN IF EXISTS category;
