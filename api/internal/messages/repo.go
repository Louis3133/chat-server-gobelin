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
	GetForUser(ctx context.Context, channel channels.Channel, user users.User, withContent bool) ([]Message, error)
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


// GetForUser renvoie les 300 derniers messages du salon (sans l'horloge interne), du plus ancien au plus récent,
// classés par date de livraison. Toutes les lettres y sont, pour que chacun voie les fusées voler ; le contenu n'est
// rempli que pour les messages de salon, les lettres que `user` a envoyées et celles qu'il a reçues une fois arrivées.
// Il est retiré par la requête elle-même : une lettre en route ou entre deux autres personnes ne quitte jamais la base
// avec son contenu. withContent à false vide tous les contenus (rafraîchissement léger).
func (r *repo) GetForUser(ctx context.Context, channel channels.Channel, user users.User, withContent bool) ([]Message, error) {
	query := `SELECT * FROM (
			SELECT
				a.id,
				CASE WHEN $3 AND (a.recipient_id IS NULL OR a.user_id = $2 OR (a.recipient_id = $2 AND a.deliver_at <= NOW()))
					THEN a.content ELSE '' END AS content,
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
				AND s.username <> 'internal-clock'
			ORDER BY a.deliver_at DESC, a.id DESC
			LIMIT 300
		) latest
		ORDER BY deliver_at, id`

	var messages []Message
	err := r.db.SelectContext(ctx, &messages, query, channel.ID, user.ID, withContent)
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
