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
	UpdatePlanet(ctx context.Context, id int64, planet Planet) (*User, error)
	UpdateProfile(ctx context.Context, id int64, profile Profile) (*User, error)
	SetTexture(ctx context.Context, id int64, data []byte) (*User, error)
	GetTexture(ctx context.Context, id int64) ([]byte, error)
	Delete(ctx context.Context, id int64) error
}

// Colonnes de l'apparence, dans l'ordre des champs de Planet, le profil, puis la version de la texture (0 sans texture)
const planetColumns = `planet_color, planet_size, planet_rings, planet_tilt, planet_ring_color, planet_ring_size, planet_spin, planet_moon,
	planet_name, pronouns, bio,
	CASE WHEN planet_texture IS NULL THEN 0 ELSE planet_texture_version END AS planet_texture_version`

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
			`+planetColumns+`
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
			`+planetColumns+`
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
		RETURNING id, username, `+planetColumns

	query = r.db.Rebind(query)
	var user User
	err := r.db.GetContext(ctx, &user, query, name, passwordHash, planet.Color, planet.Size, planet.Rings)
	if err != nil {
		return nil, fmt.Errorf("Error creating user: %w", err)
	}
	return &user, nil
}

func (r *repo) UpdatePlanet(ctx context.Context, id int64, planet Planet) (*User, error) {
	query := `UPDATE users SET
			planet_color = ?, planet_size = ?, planet_rings = ?, planet_tilt = ?,
			planet_ring_color = ?, planet_ring_size = ?, planet_spin = ?, planet_moon = ?
		WHERE id = ?
		RETURNING id, username, `+planetColumns

	query = r.db.Rebind(query)
	var user User
	err := r.db.GetContext(ctx, &user, query,
		planet.Color, planet.Size, planet.Rings, planet.Tilt, planet.RingColor, planet.RingSize, planet.Spin, planet.Moon, id)
	if err != nil {
		return nil, fmt.Errorf("Error updating planet: %w", err)
	}
	return &user, nil
}

func (r *repo) UpdateProfile(ctx context.Context, id int64, profile Profile) (*User, error) {
	query := `UPDATE users SET planet_name = ?, pronouns = ?, bio = ?
		WHERE id = ?
		RETURNING id, username, `+planetColumns

	query = r.db.Rebind(query)
	var user User
	err := r.db.GetContext(ctx, &user, query, profile.PlanetName, profile.Pronouns, profile.Bio, id)
	if err != nil {
		return nil, fmt.Errorf("Error updating profile: %w", err)
	}
	return &user, nil
}

// SetTexture remplace la texture importée (nil : la retire)
func (r *repo) SetTexture(ctx context.Context, id int64, data []byte) (*User, error) {
	query := `UPDATE users SET planet_texture = ?, planet_texture_version = planet_texture_version + 1
		WHERE id = ?
		RETURNING id, username, `+planetColumns

	query = r.db.Rebind(query)
	var user User
	err := r.db.GetContext(ctx, &user, query, data, id)
	if err != nil {
		return nil, fmt.Errorf("Error updating texture: %w", err)
	}
	return &user, nil
}

// GetTexture renvoie l'image importée, nil s'il n'y en a pas (sql.ErrNoRows si l'utilisateur n'existe pas)
func (r *repo) GetTexture(ctx context.Context, id int64) ([]byte, error) {
	var data []byte
	err := r.db.GetContext(ctx, &data, r.db.Rebind(`SELECT planet_texture FROM users WHERE id = ?`), id)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving texture: %w", err)
	}
	return data, nil
}

func (r *repo) GetAll(ctx context.Context) ([]User, error) {
  query := `SELECT
      id,
      username,
      password_hash,
      `+planetColumns+`
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
