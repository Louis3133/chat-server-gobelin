package users

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Repo interface {
	GetByUserName(ctx context.Context, username string) (*User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
	GetAll(ctx context.Context) ([]User, error)
	Create(ctx context.Context, name string, passwordHash []byte, planet Planet) (*User, error)
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

func (r *repo) GetByUserName(ctx context.Context, username string) (*User, error) {
	query := `SELECT
			id,
			username,
			password_hash,
			planet_color,
			planet_size,
			planet_rings
		FROM users
		WHERE username = ?`

	query = r.db.Rebind(query)
	var user User
	err := r.db.GetContext(ctx, &user, query, username)
	if err != nil {
		return &user, fmt.Errorf("Error retrieving users by username: %w", err)
	}
	return &user, nil
}

func (r *repo) GetByID(ctx context.Context, id int64) (*User, error) {
	query := `SELECT
			id,
			username,
			planet_color,
			planet_size,
			planet_rings
		FROM users
		WHERE id = ?`

	query = r.db.Rebind(query)
	var user User
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		return &user, fmt.Errorf("Error retrieving user by id: %w", err)
	}
	return &user, nil
}

func (r *repo) Create(ctx context.Context, name string, passwordHash []byte, planet Planet) (*User, error) {
	query := `INSERT INTO users
		(username, password_hash, creation_date, planet_color, planet_size, planet_rings) VALUES (?, ?, NOW(), ?, ?, ?)
		RETURNING id, username, planet_color, planet_size, planet_rings`

	query = r.db.Rebind(query)
	var user User
	err := r.db.GetContext(ctx, &user, query, name, passwordHash, planet.Color, planet.Size, planet.Rings)
	if err != nil {
		return nil, fmt.Errorf("Error creating user: %w", err)
	}
	return &user, nil
}

func (r *repo) GetAll(ctx context.Context) ([]User, error) {
  query := `SELECT
      id,
      username,
      password_hash,
      planet_color,
      planet_size,
      planet_rings
    FROM users`

  query = r.db.Rebind(query)
  var usersList []User
  err := r.db.SelectContext(ctx, &usersList, query)
  if err != nil {
    return nil, fmt.Errorf("Error retrieving all users: %w", err)
  }
  return usersList, nil
}


// Delete supprime l'utilisateur et son contenu (messages et user_channels sans clé étrangère : à la main)
func (r *repo) Delete(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("Error starting transaction: %w", err)
	}
	defer tx.Rollback()

	for _, query := range []string{
		`DELETE FROM messages WHERE user_id = ?`,
		`DELETE FROM user_channels WHERE user_id = ?`,
	} {
		if _, err := tx.ExecContext(ctx, tx.Rebind(query), id); err != nil {
			return fmt.Errorf("Error deleting user content: %w", err)
		}
	}

	res, err := tx.ExecContext(ctx, tx.Rebind(`DELETE FROM users WHERE id = ?`), id)
	if err != nil {
		return fmt.Errorf("Error deleting user: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}
