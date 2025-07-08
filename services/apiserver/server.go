package apiserver

import (
	"github.com/chaewonkong/matchmaker/schema"
	"github.com/labstack/echo/v4"
)

// Handler api handler
type Handler struct{}

// NewHandler creates a new API handler
func NewHandler() *Handler {
	return &Handler{}
}

// CreateTicket handles the creation of a matchmaking ticket
func (h *Handler) CreateTicket(c echo.Context) error {
	// Implementation for creating a ticket
	t := &schema.Ticket{}
	err := c.Bind(t)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}

	return nil
}

// DeleteTicketByID handles the cancellation of a matchmaking ticket by ID
func (h *Handler) DeleteTicketByID(c echo.Context) error {
	// Implementation for deleting a ticket by ID
	return nil
}

// FindAllMatchCandidates retrieves all current match candidates
func (h *Handler) FindAllMatchCandidates(c echo.Context) error {
	// Implementation for finding all match candidates
	return nil
}

// CreateOrUpdateMatchAck handles the creation or update of player acknowledgement for a match
func (h *Handler) CreateOrUpdateMatchAck(c echo.Context) error {
	// Implementation for creating or updating match acknowledgement
	return nil
}

// CreateMatchResult handles the submission of game results (win/loss)
func (h *Handler) CreateMatchResult(c echo.Context) error {
	// Implementation for creating match results
	r := &schema.MatchResult{}
	err := c.Bind(r)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}

	return nil
}

// RegisterRoutes registers the API routes with the provided Echo instance
func RegisterRoutes(e *echo.Echo, h *Handler) {
	e.POST("/tickets", h.CreateTicket)
	e.DELETE("/tickets/:ticket_id", h.DeleteTicketByID)
	e.GET("/matches/candidates", h.FindAllMatchCandidates)
	e.PUT("/matches/:match_id/ack", h.CreateOrUpdateMatchAck)
	e.POST("/match-results", h.CreateMatchResult)
}
