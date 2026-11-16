-- Texture importée (image déjà réduite par le navigateur), servie par GET /user/texture.
-- La version change à chaque envoi : l'URL de l'image change avec elle, le cache du navigateur reste juste.
ALTER TABLE users
    ADD COLUMN planet_texture BYTEA,
    ADD COLUMN planet_texture_version INTEGER NOT NULL DEFAULT 0;
