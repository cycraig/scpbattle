package handler

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

// HTTPErrorHandler renders the error.html template when an error occurs.
func HTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}

	c.Logger().Error(err)

	if code == http.StatusForbidden {
		// Don't bother rendering anything for blocked IP addresses,
		// the css files etc. get blocked anyway.
		c.HTML(code, fmt.Sprintf("%d", code))
	} else {
		c.Render(http.StatusOK, "error.html", echo.Map{
			"title": "Error",
			"error": fmt.Sprintf("%d", code),
		})
	}
}
