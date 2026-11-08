-- Lettres : un message peut être adressé à une personne et n'est lisible qu'à partir de deliver_at.
-- recipient_id reste NULL pour les messages de salon existants (général, horloge interne) : ils ne changent pas.
-- deliver_at vaut NOW() par défaut : l'ancien code d'insertion continue de fonctionner, livraison immédiate.
ALTER TABLE messages
    ADD COLUMN recipient_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    ADD COLUMN deliver_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW();

-- Boîte de lettres d'une personne : ses lettres, dans un salon, par date de livraison.
CREATE INDEX ON messages(recipient_id, channel_id, deliver_at);

-- Admin d'un salon (le soleil). NULL pour les salons créés par le système, comme « general ».
ALTER TABLE channels
    ADD COLUMN owner_id BIGINT REFERENCES users(id) ON DELETE SET NULL;

-- Distance entre deux planètes d'un même salon. Une ligne par paire, avec user_a < user_b :
-- la paire (3, 7) n'existe que sous cette forme, jamais (7, 3).
-- La distance de départ, les bornes et le rapprochement sont décidés par l'API, pas par la base.
CREATE TABLE IF NOT EXISTS relations (
    channel_id BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_a BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_b BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    distance DOUBLE PRECISION NOT NULL CHECK (distance > 0),
    exchange_count INTEGER NOT NULL DEFAULT 0,
    last_interaction TIMESTAMP WITH TIME ZONE,
    PRIMARY KEY (channel_id, user_a, user_b),
    CHECK (user_a < user_b)
);

-- Retrouver toutes les relations d'un utilisateur : la clé primaire couvre déjà user_a, pas user_b.
CREATE INDEX ON relations(channel_id, user_b);
