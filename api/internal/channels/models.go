package channels

// MaxBio est alignée sur la contrainte de la migration 00011.
const MaxBio = 500

type Channel struct {
	Name string `json:"name"`
	ID int64    `json:"id"`
	OwnerID *int64 `db:"owner_id" json:"owner_id"` // admin du salon ; nil pour ceux créés par le système (general)
	CreationDate string `db:"creation_date" json:"creation_date"`
	Bio string `db:"bio" json:"bio"`
	MemberIDs []int64 `db:"-" json:"member_ids,omitempty"` // rempli seulement par GetAllChannels (planètes des autres soleils)
}
