package birthdays

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Repo interface {
	Template(ctx context.Context) (string, error)
	SetTemplate(ctx context.Context, text string) error
	Deliver(ctx context.Context, npcID, userID int64, year int, content string) (bool, error)
}

type repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) (Repo, error) {
	return &repo{
		db: db,
	}, nil
}

func (r *repo) Template(ctx context.Context) (string, error) {
	var text string
	err := r.db.GetContext(ctx, &text, r.db.Rebind(`SELECT value FROM settings WHERE key = ?`), TemplateKey)
	if errors.Is(err, sql.ErrNoRows) {
		return DefaultTemplate, nil
	}
	if err != nil {
		return "", fmt.Errorf("Error retrieving birthday letter: %w", err)
	}
	return text, nil
}

func (r *repo) SetTemplate(ctx context.Context, text string) error {
	_, err := r.db.ExecContext(ctx, r.db.Rebind(`INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`), TemplateKey, text)
	if err != nil {
		return fmt.Errorf("Error saving birthday letter: %w", err)
	}
	return nil
}

// Deliver : une seule lettre par personne et par an, hors salon (channel_id 0), en vol pendant FlightTime
func (r *repo) Deliver(ctx context.Context, npcID, userID int64, year int, content string) (bool, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("Error starting transaction: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, tx.Rebind(`INSERT INTO birthday_letters (user_id, year) VALUES (?, ?) ON CONFLICT DO NOTHING`), userID, year)
	if err != nil {
		return false, fmt.Errorf("Error recording birthday letter: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return false, nil
	}

	_, err = tx.ExecContext(ctx, tx.Rebind(`INSERT INTO messages (content, user_id, channel_id, recipient_id, creation_date, deliver_at)
		VALUES (?, ?, 0, ?, NOW(), NOW() + make_interval(secs => ?))`), content, npcID, userID, FlightTime.Seconds())
	if err != nil {
		return false, fmt.Errorf("Error sending birthday letter: %w", err)
	}
	return true, tx.Commit()
}
