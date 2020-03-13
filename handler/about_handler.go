package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (h *Handler) AboutPageHandler(c echo.Context) error {
	return c.Render(http.StatusOK, "about.html", echo.Map{
		"title": "About",
	})
}
