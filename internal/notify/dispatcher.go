package notify

import (
	"context"
	"fmt"
	"log"

	"Flatly/internal/flat"
	"Flatly/internal/house"
	"Flatly/pkg/sender"
)

type Dispatcher struct {
	repo   house.Repository
	sender *sender.Sender
}

func NewDispatcher(repo house.Repository, s *sender.Sender) *Dispatcher {
	return &Dispatcher{repo: repo, sender: s}
}

func (d *Dispatcher) NotifyNewFlat(_ context.Context, houseID int64, item flat.Flat) {
	go func() {
		emails, err := d.repo.ListSubscriberEmails(context.Background(), houseID)
		if err != nil {
			log.Printf("notify subscribers query failed: %v", err)
			return
		}

		message := fmt.Sprintf("Новая квартира в доме %d: квартира %d, %d комнаты, цена %d", item.HouseID, item.Number, item.Rooms, item.Price)
		for _, email := range emails {
			if err := d.sender.SendEmail(context.Background(), email, message); err != nil {
				log.Printf("send notification failed: %v", err)
			}
		}
	}()
}
