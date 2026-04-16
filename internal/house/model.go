package house

import "time"

type House struct {
	ID                int64      `json:"id"`
	Address           string     `json:"address"`
	Year              int        `json:"year"`
	Developer         *string    `json:"developer,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	LastFlatCreatedAt *time.Time `json:"lastFlatCreatedAt,omitempty"`
}

type FlatSummary struct {
	ID          int64     `json:"id"`
	HouseID     int64     `json:"houseId"`
	Number      int       `json:"number"`
	Price       int64     `json:"price"`
	Rooms       int       `json:"rooms"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	CreatedBy   int64     `json:"createdBy,omitempty"`
	ModeratorID int64     `json:"moderatorId,omitempty"`
}

type CreateRequest struct {
	ID        int64   `json:"id"`
	Address   string  `json:"address"`
	Year      int     `json:"year"`
	Developer *string `json:"developer"`
}

type SubscribeRequest struct {
	Email string `json:"email"`
}

type HouseWithFlatsResponse struct {
	House House         `json:"house"`
	Flats []FlatSummary `json:"flats"`
}
