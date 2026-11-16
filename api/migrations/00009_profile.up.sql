-- Profil affiché au clic sur une planète : son nom, les pronoms et la bio de la personne.
ALTER TABLE users
    ADD COLUMN planet_name TEXT NOT NULL DEFAULT '' CHECK (char_length(planet_name) <= 40),
    ADD COLUMN pronouns TEXT NOT NULL DEFAULT '' CHECK (char_length(pronouns) <= 30),
    ADD COLUMN bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500);
