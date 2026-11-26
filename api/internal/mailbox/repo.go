package mailbox

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"chat-gobelin-api/internal/messages"

	"github.com/jmoiron/sqlx"
)

type Repo interface {
	Reply(ctx context.Context) (string, error)
	SetReply(ctx context.Context, text string) error
	Send(ctx context.Context, npcID, userID int64, content, reply string) (*messages.Message, error)
	LettersFor(ctx context.Context, npcID, userID int64) ([]messages.Message, error)
}

type repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) (Repo, error) {
	return &repo{
		db: db,
	}, nil
}

func (r *repo) Reply(ctx context.Context) (string, error) {
	var text string
	err := r.db.GetContext(ctx, &text, r.db.Rebind(`SELECT value FROM settings WHERE key = ?`), ReplyKey)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("Error retrieving npc reply: %w", err)
	}
	return text, nil
}

func (r *repo) SetReply(ctx context.Context, text string) error {
	_, err := r.db.ExecContext(ctx, r.db.Rebind(`INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`), ReplyKey, text)
	if err != nil {
		return fmt.Errorf("Error saving npc reply: %w", err)
	}
	return nil
}

// Send : la lettre part vers Zib Zib, sa réponse est programmée dans la foulée (invisible jusqu'à son départ)
func (r *repo) Send(ctx context.Context, npcID, userID int64, content, reply string) (*messages.Message, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("Error starting transaction: %w", err)
	}
	defer tx.Rollback()

	var id int64
	err = tx.GetContext(ctx, &id, tx.Rebind(`INSERT INTO messages (content, user_id, channel_id, recipient_id, creation_date, deliver_at)
		VALUES (?, ?, 0, ?, NOW(), NOW() + make_interval(secs => ?)) RETURNING id`),
		content, userID, npcID, FlightTime.Seconds())
	if err != nil {
		return nil, fmt.Errorf("Error sending npc letter: %w", err)
	}

	_, err = tx.ExecContext(ctx, tx.Rebind(`INSERT INTO messages (content, user_id, channel_id, recipient_id, creation_date, deliver_at)
		SELECT ?, ?, 0, ?, deliver_at + make_interval(secs => ?), deliver_at + make_interval(secs => ?)
		FROM messages WHERE id = ?`),
		reply, npcID, userID, ReplyDelay.Seconds(), (ReplyDelay + ReplyFlightTime).Seconds(), id)
	if err != nil {
		return nil, fmt.Errorf("Error scheduling npc reply: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("Error committing npc letter: %w", err)
	}

	var letter messages.Message
	err = r.db.GetContext(ctx, &letter, r.db.Rebind(lettersQuery+` AND m.id = ?`), userID, npcID, userID, userID, npcID, id)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving npc letter: %w", err)
	}
	return &letter, nil
}

// Lettres échangées avec Zib Zib : une réponse n'apparaît qu'à son départ, son contenu qu'à son arrivée
const lettersQuery = `SELECT
		m.id,
		CASE WHEN m.deliver_at <= NOW() OR m.user_id = ? THEN m.content ELSE '' END AS content,
		m.user_id,
		s.username AS user,
		m.recipient_id,
		rc.username AS recipient,
		m.creation_date,
		m.deliver_at,
		(m.deliver_at > NOW()) AS in_transit,
		m.rocket_removed
	FROM messages m
	JOIN users s ON s.id = m.user_id
	JOIN users rc ON rc.id = m.recipient_id
	WHERE m.channel_id = 0 AND m.creation_date <= NOW()
		AND ((m.user_id = ? AND m.recipient_id = ?) OR (m.user_id = ? AND m.recipient_id = ?))`

func (r *repo) LettersFor(ctx context.Context, npcID, userID int64) ([]messages.Message, error) {
	letters := []messages.Message{}
	err := r.db.SelectContext(ctx, &letters, r.db.Rebind(lettersQuery+` ORDER BY m.deliver_at, m.id`), userID, npcID, userID, userID, npcID)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving npc letters: %w", err)
	}
	return letters, nil
}
