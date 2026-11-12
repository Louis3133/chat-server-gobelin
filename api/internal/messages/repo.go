package messages

import (
	"context"
	"database/sql"
	"fmt"

	"chat-gobelin-api/internal/users"
	"chat-gobelin-api/internal/channels"

	"github.com/jmoiron/sqlx"
)

type Repo interface {
	GetForUser(ctx context.Context, channel channels.Channel, user users.User) ([]Message, error)
	Create(ctx context.Context, content string, channel channels.Channel, user users.User) (*Message, error)
	RemoveRocket(ctx context.Context, id int64, recipient users.User) error
}

type repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) (Repo, error) {
	return &repo{
		db: db,
	}, nil
}


// GetForUser renvoie les 100 derniers messages que `user` a le droit de voir dans le salon, du plus ancien au plus
// récent, classés par date de livraison :
//   - les messages adressés à tout le salon ;
//   - les lettres qu'il a envoyées (avec leur contenu, même en route) ;
//   - les lettres qui lui sont adressées : arrivées avec leur contenu, en route sans (Content vide, InTransit vrai).
// Les lettres entre d'autres personnes n'apparaissent jamais. Le contenu d'une lettre en route est retiré par la
// requête elle-même : il ne quitte pas la base.
func (r *repo) GetForUser(ctx context.Context, channel channels.Channel, user users.User) ([]Message, error) {
	query := `SELECT * FROM (
			SELECT
				a.id,
				CASE WHEN a.deliver_at > NOW() AND a.user_id <> $2 THEN '' ELSE a.content END AS content,
				a.user_id,
				s.username AS user,
				a.recipient_id,
				rc.username AS recipient,
				a.creation_date,
				a.deliver_at,
				(a.deliver_at > NOW()) AS in_transit,
				a.rocket_removed
			FROM messages a
			JOIN users s ON s.id = a.user_id
			LEFT JOIN users rc ON rc.id = a.recipient_id
			WHERE a.channel_id = $1
				AND (a.recipient_id IS NULL OR a.user_id = $2 OR a.recipient_id = $2)
			ORDER BY a.deliver_at DESC, a.id DESC
			LIMIT 100
		) latest
		ORDER BY deliver_at, id`

	var messages []Message
	err := r.db.SelectContext(ctx, &messages, query, channel.ID, user.ID)
	if err != nil {
		return messages, fmt.Errorf("Error retrieving messages: %w", err)
	}
	return messages, nil
}

func (r *repo) Create(ctx context.Context, content string, channel channels.Channel, user users.User) (*Message, error) {
	query := `INSERT INTO messages
		(content, user_id, channel_id, creation_date) VALUES (?, ?, ?, NOW())
		RETURNING id, content, user_id, creation_date, deliver_at`

	query = r.db.Rebind(query)
	var message Message
	err := r.db.GetContext(ctx, &message, query, content, user.ID, channel.ID)
	if err != nil {
		return nil, fmt.Errorf("Error creating channel: %w", err)
	}
	message.User = user.Username
	return &message, nil
}

// RemoveRocket retire la fusée d'une lettre arrivée, seulement pour son destinataire (sql.ErrNoRows sinon)
func (r *repo) RemoveRocket(ctx context.Context, id int64, recipient users.User) error {
	res, err := r.db.ExecContext(ctx, r.db.Rebind(`UPDATE messages SET rocket_removed = TRUE
		WHERE id = ? AND recipient_id = ? AND deliver_at <= NOW()`), id, recipient.ID)
	if err != nil {
		return fmt.Errorf("Error removing rocket: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
