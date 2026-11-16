-- 1. Un utilisateur n'est membre d'un salon qu'une fois : avant, POST /channel/enter répété ajoutait une ligne
--    à chaque appel, donc une planète en double dans le système solaire.
DELETE FROM user_channels a
    USING user_channels b
    WHERE a.ctid < b.ctid
      AND a.user_id = b.user_id
      AND a.channel_id = b.channel_id;

CREATE UNIQUE INDEX IF NOT EXISTS user_channels_user_channel_key ON user_channels(user_id, channel_id);

-- 2. Correctif de 00002 : ADD COLUMN ... DEFAULT NOW() a donné à tous les messages déjà présents la date de la
--    migration comme deliver_at. Un message de salon est livré à sa création : on remet sa vraie date,
--    sinon le tri par date de livraison mélangerait l'historique.
UPDATE messages SET deliver_at = creation_date WHERE recipient_id IS NULL;
