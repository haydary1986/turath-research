-- Removes the footer research tools list.
UPDATE site_content SET content = content - 'tools' WHERE id = 1;
