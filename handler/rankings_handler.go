package handler

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

// Candidate wraps SCP fields to be rendered in the rankings.html template.
type Candidate struct {
	Rank   int
	Name   string
	Desc   string
	Link   string
	Rating int64
	Wins   uint64
	Losses uint64
}

// RankingsPageHandler renders the rankings.html template.
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
	return c.Render(http.StatusOK, "rankings.html", echo.Map{
		"title":      "Rankings",
		"main-image": h.imageDir + rankedSCPs[0].Image,
		"candidates": candidates,
	})
}
