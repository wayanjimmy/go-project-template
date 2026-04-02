-- migrate:up
ALTER TABLE users
ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';

-- migrate:down
ALTER TABLE users
DROP COLUMN IF EXISTS status;
