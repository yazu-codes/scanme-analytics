package repositories

import (
	"context"
	"time"

	db "github.com/yazu-codes/scanme-analytics.git/src/database"
	"github.com/yazu-codes/scanme-analytics.git/src/models"
)

type EventsRepository struct {
	db *db.DB
}

func NewEventsRepository(db *db.DB) *EventsRepository {
	return &EventsRepository{db: db}
}

func (r *EventsRepository) Create(event models.Event) error {
	result := r.db.Connection.Create(&event)
	return result.Error
}

func (r *EventsRepository) EventsSince(ctx context.Context, since time.Time) ([]models.Event, error) {
	var events []models.Event
	result := r.db.Connection.Where("created_at >= ?", since).Find(&events)
	if result.Error != nil {
		return nil, result.Error
	}
	return events, nil
}
