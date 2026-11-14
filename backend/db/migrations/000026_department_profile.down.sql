-- Removes the department profile block.
UPDATE site_content SET content = content - 'department' WHERE id = 1;
