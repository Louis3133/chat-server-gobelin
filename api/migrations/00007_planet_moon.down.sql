ALTER TABLE users
    DROP CONSTRAINT IF EXISTS planet_rings_or_moon,
    DROP COLUMN IF EXISTS planet_moon;
