package handler

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

type Candidate struct {
	Rank   int
	Name   string
	Desc   string
	Link   string
	Rating int64
	Wins   uint64
	Losses uint64
}

func (h *Handler) RankingsPageHandler(c echo.Context) error {
	rankedSCPs, err := h.scpCache.GetRankedSCPs()
	if err != nil {
		msg := fmt.Sprintf("Error retrieving ranked SCPs")
		c.Logger().Error(msg, err)
		return echo.NewHTTPError(http.StatusInternalServerError, msg)
	}
	// TODO: use the rankedSCPs slice directly to avoid extra allocation per request?
	candidates := make([]Candidate, len(rankedSCPs))
	for i, scp := range rankedSCPs {
		candidates[i] = Candidate{
			Rank:   i + 1,
			Name:   scp.Name,
			Desc:   scp.Description,
			Link:   scp.Link,
			Rating: int64(scp.Rating),
			Wins:   scp.Wins,
			Losses: scp.Losses,
		}
	}
	IMAGE_DIR := "images/"
	return c.Render(http.StatusOK, "rankings.html", echo.Map{
		"title":      "Rankings",
		"main-image": IMAGE_DIR + rankedSCPs[0].Image,
		"candidates": candidates,
	})
}
