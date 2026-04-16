package flat

import "time"

type Status string

const (
	StatusCreated      Status = "created"
	StatusApproved     Status = "approved"
	StatusDeclined     Status = "declined"
	StatusOnModeration Status = "on moderation"
)

type Flat struct {
	ID          int64     `json:"id"`
	HouseID     int64     `json:"houseId"`
	Number      int       `json:"number"`
	Price       int64     `json:"price"`
	Rooms       int       `json:"rooms"`
	Status      Status    `json:"status"`
	CreatedBy   int64     `json:"createdBy,omitempty"`
	ModeratorID *int64    `json:"moderatorId,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CreateRequest struct {
	HouseID int64 `json:"houseId"`
	Number  int   `json:"number"`
	Price   int64 `json:"price"`
	Rooms   int   `json:"rooms"`
}

type UpdateRequest struct {
	ID     int64  `json:"id"`
	Status Status `json:"status"`
}
