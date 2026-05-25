package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ParticipantMiddleware checks for a participant_id in the X-Participant-ID header.
// If absent, it generates a new UUID and stores it in the request context.
func ParticipantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		participantID := c.GetHeader("X-Participant-ID")

		// Validate the UUID format
		if participantID != "" {
			if _, err := uuid.Parse(participantID); err != nil {
				// Invalid UUID, generate a new one
				participantID = ""
			}
		}

		// If there is no valid participant_id, generate a new one
		if participantID == "" {
			participantID = uuid.New().String()
		}

		// Store it in the context
		c.Set("participant_id", participantID)

		// Echo it back to the client in the response header
		c.Header("X-Participant-ID", participantID)

		c.Next()
	}
}
