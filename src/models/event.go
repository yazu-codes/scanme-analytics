package models

import (
	"fmt"

	"github.com/google/uuid"
)

type Event struct {
	ID                    uint      `gorm:"primaryKey" json:"id"`
	ClientID              uuid.UUID `gorm:"type:uuid;not null" json:"client_id"`
	Name                  string    `gorm:"not null" json:"name"`
	CorrespondingItemName string    `json:"corresponding_item_name"`
	CreatedAt             int64     `gorm:"autoCreateTime"`
	IpHash                string    `gorm:"not null" json:"-"`
}

// todo: Add a method to validate the event data before saving it to the database.
// This could include checking that the ClientID is a valid UUID, that the Name is not empty,
// and that the IpHash is a valid hash. also have a permitted event names list and check if the
// event name is in that list. If not, return an error. This will help prevent invalid or malicious
// data from being saved to the database.

func (e *Event) Validate() error {
	// Check if ClientID is a valid UUID
	if e.ClientID == uuid.Nil {
		return fmt.Errorf("invalid ClientID: cannot be nil")
	}
	// Check if Name is not empty
	if e.Name == "" || !isPermittedEventName(e.Name) {
		return fmt.Errorf("invalid Name: cannot be empty or not permitted")
	}
	// Check if IpHash is not empty
	if e.IpHash == "" {
		return fmt.Errorf("invalid IpHash: cannot be empty")
	}
	return nil
}

func isPermittedEventName(name string) bool {
	permittedEventNames := []string{
		"click",
		"view",
		"code_scan_review",
		"code_scan",
		"qr_scan",
		"item_click",
	}
	for _, permittedName := range permittedEventNames {
		if name == permittedName {
			return true
		}
	}
	return false
}
