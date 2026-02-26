ALTER TABLE users ADD CONSTRAINT chk_email CHECK (email ~* '^.+@.+$');
