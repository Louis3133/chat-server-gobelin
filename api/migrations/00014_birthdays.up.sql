-- Anniversaire (jour et mois, privé) et lettres d'anniversaire de Zib Zib : une par personne et par an.
ALTER TABLE users
    ADD COLUMN birth_month SMALLINT CHECK (birth_month BETWEEN 1 AND 12),
    ADD COLUMN birth_day SMALLINT CHECK (birth_day BETWEEN 1 AND 31),
    ADD CONSTRAINT users_birthday_complete CHECK ((birth_month IS NULL) = (birth_day IS NULL));

CREATE TABLE birthday_letters (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    year INTEGER NOT NULL,
    PRIMARY KEY (user_id, year)
);

CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
