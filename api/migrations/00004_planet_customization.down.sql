ALTER TABLE users
    DROP COLUMN IF EXISTS planet_color,
    DROP COLUMN IF EXISTS planet_size,
    DROP COLUMN IF EXISTS planet_rings;
