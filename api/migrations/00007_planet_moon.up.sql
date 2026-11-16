-- Petite lune autour de la planète, à la place des anneaux : l'un ou l'autre, jamais les deux.
ALTER TABLE users
    ADD COLUMN planet_moon BOOLEAN NOT NULL DEFAULT FALSE,
    ADD CONSTRAINT planet_rings_or_moon CHECK (NOT (planet_rings AND planet_moon));
