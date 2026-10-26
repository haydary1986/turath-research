-- Link a user to their Google account (the stable "sub" claim, not the email).
ALTER TABLE users ADD COLUMN google_sub VARCHAR(64) UNIQUE;
