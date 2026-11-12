-- Fusée retirée de la planète par son destinataire, pour faire de la place : la lettre, elle, reste.
ALTER TABLE messages ADD COLUMN rocket_removed BOOLEAN NOT NULL DEFAULT FALSE;
