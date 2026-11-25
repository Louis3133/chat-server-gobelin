-- Mot de passe facultatif d'un salon (hash bcrypt) : demandé une seule fois, à l'entrée.
ALTER TABLE channels ADD COLUMN password_hash BYTEA;
