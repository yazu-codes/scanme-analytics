package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yazu-codes/scanme-analytics.git/src/models"
	"github.com/yazu-codes/scanme-analytics.git/src/repositories"
)

type EventHandler struct {
	eventRepo *repositories.EventsRepository
}

func NewEventHandler(eventRepo *repositories.EventsRepository) *EventHandler {
	return &EventHandler{
		eventRepo: eventRepo,
	}
}

func (eh *EventHandler) RecordEvent(c *gin.Context) {
	ipHash, exists := c.Get("iphash")
	if !exists {
		c.AbortWithStatusJSON(500, gin.H{"error": "Failed to retrieve IP hash"})
		return
	}

	var event models.Event
	event.IpHash = ipHash.(string)

	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if event.Validate() != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event data"})
		return
	}

	err := eh.eventRepo.Create(event)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": "Failed to record event"})
		return
	}

	c.JSON(200, gin.H{"message": "Event recorded successfully"})
	return
}

const maxDays = 180

func (eh *EventHandler) GetEvents(c *gin.Context) {
	days, err := strconv.Atoi(c.DefaultQuery("days", "7"))
	if err != nil || days < 1 || days > maxDays {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("days must be an integer between 1 and %d", maxDays),
		})
		return
	}

	since := time.Now().AddDate(0, 0, -days)

	events, err := eh.eventRepo.EventsSince(c.Request.Context(), since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"days":   days,
		"since":  since,
		"count":  len(events),
		"events": events,
	})
}
