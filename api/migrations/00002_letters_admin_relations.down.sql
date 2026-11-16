DROP TABLE IF EXISTS relations;

ALTER TABLE channels DROP COLUMN IF EXISTS owner_id;

DROP INDEX IF EXISTS messages_recipient_id_channel_id_deliver_at_idx;
ALTER TABLE messages
    DROP COLUMN IF EXISTS deliver_at,
    DROP COLUMN IF EXISTS recipient_id;
