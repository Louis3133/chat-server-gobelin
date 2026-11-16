-- Apparence complète de la planète, choisie après l'inscription et modifiable ensuite (PUT /me/planet).
ALTER TABLE users
    ADD COLUMN planet_tilt DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (planet_tilt BETWEEN -1.2 AND 1.2),
    ADD COLUMN planet_ring_color TEXT NOT NULL DEFAULT '#d9c9a3' CHECK (planet_ring_color ~ '^#[0-9a-fA-F]{6}$'),
    ADD COLUMN planet_ring_size DOUBLE PRECISION NOT NULL DEFAULT 2 CHECK (planet_ring_size BETWEEN 1.5 AND 4),
    ADD COLUMN planet_spin DOUBLE PRECISION NOT NULL DEFAULT 0.01 CHECK (planet_spin BETWEEN -0.1 AND 0.1);
