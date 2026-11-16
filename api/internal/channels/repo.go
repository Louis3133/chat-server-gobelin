package channels

import (
	"context"
	"database/sql"
	"fmt"

	"chat-gobelin-api/internal/users"

	"github.com/jmoiron/sqlx"
)

type Repo interface {
	GetByUser(ctx context.Context, user users.User) ([]Channel, error)
	GetByID(ctx context.Context, id int64) (*Channel, error)
	GetByName(ctx context.Context, name string) (*Channel, error)
	GetAllChannels(ctx context.Context) ([]Channel, error)
	Create(ctx context.Context, name string, ownerID *int64) (*Channel, error)
	AddUserToChannel(ctx context.Context, channel Channel, user users.User) error
	IsMember(ctx context.Context, channelID int64, userID int64) (bool, error)
	UpdateBio(ctx context.Context, id int64, bio string) (*Channel, error)
	Delete(ctx context.Context, id int64) error
}

type repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) (Repo, error) {
	return &repo{
		db: db,
	}, nil
}

func (r *repo) GetByUser(ctx context.Context, user users.User) ([]Channel, error) {
	query := `SELECT
			a.id,
			a.name,
			a.owner_id,
			a.creation_date,
			a.bio
		FROM channels a
		JOIN user_channels b ON a.id = b.channel_id
		WHERE b.user_id = ?`

	query = r.db.Rebind(query)
	var channels []Channel
	err := r.db.SelectContext(ctx, &channels, query, user.ID)
	if err != nil {
		return channels, fmt.Errorf("Error retrieving channels: %w", err)
	}
	return channels, nil
}

func (r *repo) GetAllChannels(ctx context.Context) ([]Channel, error) {
	query := `SELECT
			id,
			name,
			owner_id,
			bio
		FROM channels`

	query = r.db.Rebind(query)
	var channels []Channel
	err := r.db.SelectContext(ctx, &channels, query)
	if err != nil {
		return channels, fmt.Errorf("Error retrieving channels: %w", err)
	}

	var members []struct {
		ChannelID int64 `db:"channel_id"`
		UserID    int64 `db:"user_id"`
	}
	err = r.db.SelectContext(ctx, &members, `SELECT channel_id, user_id FROM user_channels ORDER BY user_id`)
	if err != nil {
		return channels, fmt.Errorf("Error retrieving channel members: %w", err)
	}
	byChannel := make(map[int64][]int64)
	for _, m := range members {
		byChannel[m.ChannelID] = append(byChannel[m.ChannelID], m.UserID)
	}
	for i := range channels {
		channels[i].MemberIDs = byChannel[channels[i].ID]
	}
	return channels, nil
}

func (r *repo) GetByName(ctx context.Context, name string) (*Channel, error) {
	query := `SELECT
			id,
			name,
			owner_id,
			creation_date,
			bio
		FROM channels
		WHERE name = ?`

	query = r.db.Rebind(query)
	var channel Channel
	err := r.db.GetContext(ctx, &channel, query, name)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving channels: %w", err)
	}
	return &channel, nil
}

func (r *repo) GetByID(ctx context.Context, id int64) (*Channel, error) {
	query := `SELECT
			id,
			name,
			owner_id,
			creation_date,
			bio
		FROM channels
		WHERE id = ?`

	query = r.db.Rebind(query)
	var channel Channel
	err := r.db.GetContext(ctx, &channel, query, id)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving channels: %w", err)
	}
	return &channel, nil
}

func (r *repo) Create(ctx context.Context, name string, ownerID *int64) (*Channel, error) {
	query := `INSERT INTO channels
		(name, owner_id, creation_date) VALUES (?, ?, NOW())
		RETURNING id, name, owner_id, creation_date, bio`

	query = r.db.Rebind(query)
	var channel Channel
	err := r.db.GetContext(ctx, &channel, query, name, ownerID)
	if err != nil {
		return nil, fmt.Errorf("Error creating channel: %w", err)
	}

	//TODO: add creator to it
	return &channel, nil
}

// AddUserToChannel est idempotent : entrer deux fois dans le même salon ne crée pas de doublon.
func (r *repo) AddUserToChannel(ctx context.Context, channel Channel, user users.User) error {
	query := `INSERT INTO user_channels
		(user_id, channel_id) VALUES (?, ?)
		ON CONFLICT (user_id, channel_id) DO NOTHING`

	query = r.db.Rebind(query)
	_, err := r.db.ExecContext(ctx, query, user.ID, channel.ID)
	if err != nil {
		return fmt.Errorf("Error adding user to channel: %w", err)
	}
	return nil
}

func (r *repo) IsMember(ctx context.Context, channelID int64, userID int64) (bool, error) {
	query := `SELECT EXISTS (
		SELECT 1 FROM user_channels WHERE channel_id = ? AND user_id = ?
	)`

	query = r.db.Rebind(query)
	var member bool
	err := r.db.GetContext(ctx, &member, query, channelID, userID)
	if err != nil {
		return false, fmt.Errorf("Error checking membership: %w", err)
	}
	return member, nil
}

func (r *repo) UpdateBio(ctx context.Context, id int64, bio string) (*Channel, error) {
	query := `UPDATE channels SET bio = ? WHERE id = ?
		RETURNING id, name, owner_id, creation_date, bio`

	query = r.db.Rebind(query)
	var channel Channel
	err := r.db.GetContext(ctx, &channel, query, bio, id)
	if err != nil {
		return nil, fmt.Errorf("Error updating channel bio: %w", err)
	}
	return &channel, nil
}

// Delete supprime le salon et son contenu (messages et user_channels sans clé étrangère : à la main)
func (r *repo) Delete(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("Error starting transaction: %w", err)
	}
	defer tx.Rollback()

	for _, query := range []string{
		`DELETE FROM messages WHERE channel_id = ?`,
		`DELETE FROM user_channels WHERE channel_id = ?`,
	} {
		if _, err := tx.ExecContext(ctx, tx.Rebind(query), id); err != nil {
			return fmt.Errorf("Error deleting channel content: %w", err)
		}
	}

	res, err := tx.ExecContext(ctx, tx.Rebind(`DELETE FROM channels WHERE id = ?`), id)
	if err != nil {
		return fmt.Errorf("Error deleting channel: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}
