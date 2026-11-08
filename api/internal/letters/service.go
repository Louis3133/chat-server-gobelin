package letters

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"chat-gobelin-api/internal/channels"
	"chat-gobelin-api/internal/messages"
	"chat-gobelin-api/internal/users"

	"github.com/jmoiron/sqlx"
)

var (
	ErrSelfLetter         = errors.New("cannot send a letter to yourself")
	ErrNotMember          = errors.New("sender is not a member of the channel")
	ErrRecipientNotMember = errors.New("recipient is not a member of the channel")
)

// Service envoie les lettres et décrit le système solaire d'un salon. Il est seul à écrire dans `relations`.
type Service struct {
	db    *sqlx.DB
	rules Rules
}

func NewService(db *sqlx.DB, rules Rules) *Service {
	return &Service{db: db, rules: rules}
}

// Send envoie une lettre à une personne du salon. Tout se passe dans une transaction : la distance du couple
// est verrouillée, mise à jour si c'est une réponse, puis la lettre est enregistrée avec sa date de livraison.
//
// Une lettre est une réponse quand le dernier message échangé entre les deux (dans ce salon) venait de
// l'autre et qu'il est déjà arrivé. Une réponse rapproche d'un cran (Rules.Approached), et sa propre durée de
// trajet se calcule avec la distance déjà réduite. Écrire sans réponse, ou plusieurs lettres de suite,
// ne rapproche pas.
func (s *Service) Send(ctx context.Context, channelID int64, sender users.User, recipientID int64, content string) (*messages.Message, error) {
	if sender.ID == recipientID {
		return nil, ErrSelfLetter
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("Error starting transaction: %w", err)
	}
	defer tx.Rollback()

	if ok, err := isMember(ctx, tx, channelID, sender.ID); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrNotMember
	}
	if ok, err := isMember(ctx, tx, channelID, recipientID); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrRecipientNotMember
	}

	// Une seule ligne par couple, rangé dans l'ordre (user_a < user_b).
	a, b := sender.ID, recipientID
	if a > b {
		a, b = b, a
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO relations (channel_id, user_a, user_b, distance) VALUES ($1, $2, $3, $4)
		 ON CONFLICT DO NOTHING`,
		channelID, a, b, s.rules.InitialDistance())
	if err != nil {
		return nil, fmt.Errorf("Error creating relation: %w", err)
	}

	// FOR UPDATE : deux lettres simultanées du même couple passent l'une après l'autre.
	var distance float64
	err = tx.GetContext(ctx, &distance,
		`SELECT distance FROM relations WHERE channel_id = $1 AND user_a = $2 AND user_b = $3 FOR UPDATE`,
		channelID, a, b)
	if err != nil {
		return nil, fmt.Errorf("Error reading relation: %w", err)
	}

	reply, err := isReply(ctx, tx, channelID, sender.ID, recipientID)
	if err != nil {
		return nil, err
	}
	if reply {
		distance = s.rules.Approached(distance)
		_, err = tx.ExecContext(ctx,
			`UPDATE relations SET distance = $4, exchange_count = exchange_count + 1, last_interaction = NOW()
			 WHERE channel_id = $1 AND user_a = $2 AND user_b = $3`,
			channelID, a, b, distance)
	} else {
		_, err = tx.ExecContext(ctx,
			`UPDATE relations SET last_interaction = NOW() WHERE channel_id = $1 AND user_a = $2 AND user_b = $3`,
			channelID, a, b)
	}
	if err != nil {
		return nil, fmt.Errorf("Error updating relation: %w", err)
	}

	// La date de livraison est calculée par la base (NOW() + délai) : c'est la même horloge que celle qui
	// décide, à la lecture, si la lettre est arrivée.
	delay := s.rules.Delay(distance)
	var letter messages.Message
	err = tx.GetContext(ctx, &letter,
		`INSERT INTO messages (content, user_id, channel_id, recipient_id, creation_date, deliver_at)
		 VALUES ($1, $2, $3, $4, NOW(), NOW() + make_interval(secs => $5::double precision))
		 RETURNING id, content, user_id, recipient_id, creation_date, deliver_at`,
		content, sender.ID, channelID, recipientID, delay.Seconds())
	if err != nil {
		return nil, fmt.Errorf("Error creating letter: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("Error committing letter: %w", err)
	}
	letter.User = sender.Username
	letter.InTransit = delay > 0
	return &letter, nil
}

func isMember(ctx context.Context, q sqlx.QueryerContext, channelID, userID int64) (bool, error) {
	var ok bool
	err := sqlx.GetContext(ctx, q, &ok,
		`SELECT EXISTS (SELECT 1 FROM user_channels WHERE channel_id = $1 AND user_id = $2)`, channelID, userID)
	if err != nil {
		return false, fmt.Errorf("Error checking membership: %w", err)
	}
	return ok, nil
}

func isReply(ctx context.Context, q sqlx.QueryerContext, channelID, senderID, recipientID int64) (bool, error) {
	var last struct {
		UserID    int64 `db:"user_id"`
		Delivered bool  `db:"delivered"`
	}
	err := sqlx.GetContext(ctx, q, &last,
		`SELECT user_id, deliver_at <= NOW() AS delivered
		 FROM messages
		 WHERE channel_id = $1
		   AND ((user_id = $2 AND recipient_id = $3) OR (user_id = $3 AND recipient_id = $2))
		 ORDER BY id DESC
		 LIMIT 1`,
		channelID, senderID, recipientID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("Error reading last letter: %w", err)
	}
	return last.UserID == recipientID && last.Delivered, nil
}

// Planet est un membre du salon, vu par celui qui regarde : sa distance à lui et le temps qu'une lettre mettrait.
type Planet struct {
	ID           int64    `json:"id"`
	Username     string   `json:"username"`
	Me           bool     `json:"me"`
	Distance     *float64 `json:"distance"`      // nil pour soi-même
	DelaySeconds *float64 `json:"delay_seconds"` // temps réel d'une lettre vers cette personne, maintenant
	Exchanges    int      `json:"exchanges"`     // réponses échangées avec elle
	Color        string   `json:"color"`         // apparence choisie à l'inscription
	Size         float64  `json:"size"`
	Rings        bool     `json:"rings"`
}

type System struct {
	Channel     channels.Channel `json:"channel"` // le soleil
	Planets     []Planet         `json:"planets"`
	MinDistance float64          `json:"min_distance"`
	MaxDistance float64          `json:"max_distance"`
	TimeScale   float64          `json:"time_scale"`
}

// System décrit le salon du point de vue de `me` : seules SES distances sont renvoyées, pas celles des autres
// couples (qui parle à qui reste privé).
func (s *Service) System(ctx context.Context, channel channels.Channel, me int64) (*System, error) {
	var rows []struct {
		ID        int64   `db:"id"`
		Username  string  `db:"username"`
		Distance  float64 `db:"distance"`
		Exchanges int     `db:"exchanges"`
		Color     string  `db:"planet_color"`
		Size      float64 `db:"planet_size"`
		Rings     bool    `db:"planet_rings"`
	}
	err := s.db.SelectContext(ctx, &rows,
		`SELECT u.id, u.username, u.planet_color, u.planet_size, u.planet_rings,
		        COALESCE(r.distance, $3::double precision) AS distance,
		        COALESCE(r.exchange_count, 0) AS exchanges
		 FROM user_channels uc
		 JOIN users u ON u.id = uc.user_id
		 LEFT JOIN relations r ON r.channel_id = uc.channel_id
		      AND r.user_a = LEAST(u.id, $2::bigint) AND r.user_b = GREATEST(u.id, $2::bigint)
		 WHERE uc.channel_id = $1
		 ORDER BY u.id`,
		channel.ID, me, s.rules.InitialDistance())
	if err != nil {
		return nil, fmt.Errorf("Error retrieving system: %w", err)
	}

	planets := make([]Planet, 0, len(rows))
	for _, row := range rows {
		p := Planet{ID: row.ID, Username: row.Username, Me: row.ID == me, Color: row.Color, Size: row.Size, Rings: row.Rings}
		if !p.Me {
			distance := row.Distance
			delay := s.rules.Delay(distance).Seconds()
			p.Distance, p.DelaySeconds, p.Exchanges = &distance, &delay, row.Exchanges
		}
		planets = append(planets, p)
	}
	return &System{
		Channel:     channel,
		Planets:     planets,
		MinDistance: s.rules.MinDistance,
		MaxDistance: s.rules.MaxDistance,
		TimeScale:   s.rules.TimeScale,
	}, nil
}
