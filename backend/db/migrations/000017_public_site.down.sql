-- Announcement attachments live in portal_files; remove them with the table.
DELETE FROM portal_files WHERE role = 'announcement';
DROP TABLE IF EXISTS announcements;
DROP TABLE IF EXISTS site_content;
