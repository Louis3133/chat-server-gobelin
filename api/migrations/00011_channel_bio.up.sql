-- Bio du salon, affichée dans la modale au clic sur son soleil.
ALTER TABLE channels ADD COLUMN bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500);
