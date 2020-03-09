package handler

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

func HTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}
	// errorPage := path.Join("static", fmt.Sprintf("%d.html", code))
	// if err := c.File(errorPage); err != nil {
	// 	c.Logger().Error(err)
	// }
	c.Logger().Error(err)
	c.Render(http.StatusOK, "error.html", echo.Map{
		"title": "Error",
		"error": fmt.Sprintf("%d", code),
	})
}
