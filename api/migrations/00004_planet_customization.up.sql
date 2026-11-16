-- Personnalisation de la planète, choisie à l'inscription (non modifiable ensuite) : couleur libre, taille, anneaux.
ALTER TABLE users
    ADD COLUMN planet_color TEXT NOT NULL DEFAULT '#8d9db6' CHECK (planet_color ~ '^#[0-9a-fA-F]{6}$'),
    ADD COLUMN planet_size DOUBLE PRECISION NOT NULL DEFAULT 0.55 CHECK (planet_size BETWEEN 0.4 AND 1.0),
    ADD COLUMN planet_rings BOOLEAN NOT NULL DEFAULT FALSE;

-- Les comptes déjà créés reçoivent une couleur et une taille stables, tirées de leur id (sinon toutes grises).
UPDATE users SET
    planet_color = (ARRAY['#4fc3f7', '#ef5350', '#66bb6a', '#ffca28', '#ab47bc', '#ff7043', '#26c6da', '#ec407a'])[(id % 8) + 1],
    planet_size = 0.45 + (id % 5) * 0.1;
