-- Distance du couple avant cette lettre, si c'était une réponse : l'annuler en route rend le rapprochement.
ALTER TABLE messages ADD COLUMN approached_from DOUBLE PRECISION;
