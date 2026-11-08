package messages

type Message struct {
	ID           int64  `json:"-"`
	Content      string `json:"content"`
	UserID       int64  `db:"user_id" json:"user_id"`
	User         string `json:"user"`
	CreationDate string `db:"creation_date" json:"creation_date"`

	// Lettres : RecipientID est nil pour un message adressé à tout le salon. Tant qu'une lettre est en route
	// (InTransit), son destinataire n'en reçoit que ces métadonnées : Content est vide.
	RecipientID *int64  `db:"recipient_id" json:"recipient_id"`
	Recipient   *string `db:"recipient" json:"recipient"`
	DeliverAt   string  `db:"deliver_at" json:"deliver_at"`
	InTransit   bool    `db:"in_transit" json:"in_transit"`
}
