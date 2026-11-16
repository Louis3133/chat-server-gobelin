ALTER TABLE users
    DROP COLUMN IF EXISTS planet_tilt,
    DROP COLUMN IF EXISTS planet_ring_color,
    DROP COLUMN IF EXISTS planet_ring_size,
    DROP COLUMN IF EXISTS planet_spin;
