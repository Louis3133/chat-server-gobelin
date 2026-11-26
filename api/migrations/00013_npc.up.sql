-- PNJ : un compte que personne ne peut ouvrir (Zib Zib), sa planète est réglée par les admins.
ALTER TABLE users ADD COLUMN is_npc BOOLEAN NOT NULL DEFAULT false;
