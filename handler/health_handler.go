package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// HealthCheck is a simple JSON-encoded response to API health-checks.
type HealthCheck struct {
	Status string `json:"status"`
}

// HealthCheckHandler processes health-check GET requests.
func (h *Handler) HealthCheckHandler(c echo.Context) error {
	// Heath check RFC:
	// https://tools.ietf.org/html/draft-inadarei-api-health-check-04#section-3

	// TODO: db.Ping() the database to check if the connection is still healthy
	resp := new(HealthCheck)
	resp.Status = "ok"
	return c.JSON(http.StatusOK, resp)
}
