package handler

import (
	"fmt"
	"math"
	"net/http"
	"sync"

	"github.com/cycraig/scpbattle/model"
	"github.com/labstack/echo/v4"
)

func (h *Handler) VotePageHandler(c echo.Context) error {
	randomSCPs, err := h.scpCache.GetRandomSCPs(2)
	if err != nil {
		msg := "Error retrieving random SCPs "
		c.Logger().Error(msg, err)
		return echo.NewHTTPError(http.StatusInternalServerError, msg)
	}
	if len(randomSCPs) < 2 {
		// Shouldn't happen
		msg := fmt.Sprintf("Too few random SCPs retrieved: %d", len(randomSCPs))
		c.Logger().Error(msg)
		return echo.NewHTTPError(http.StatusInternalServerError, msg)
	}
	left := randomSCPs[0]
	right := randomSCPs[1]
	imageDir := "images/"
	return c.Render(http.StatusOK, "vote.html", echo.Map{
		"title":      "Vote",
		"id_left":    left.ID,
		"name_left":  left.Name,
		"desc_left":  left.Description,
		"img_left":   imageDir + left.Image,
		"link_left":  left.Link,
		"id_right":   right.ID,
		"name_right": right.Name,
		"desc_right": right.Description,
		"img_right":  imageDir + right.Image,
		"link_right": right.Link,
	})
}

type VoteRequest struct {
	// Using platform-dependent types is concerning, I wonder why gorm defaults to uint instead of uint64...
	// We'll never go above 2^32-1 SCPs anyway, so it doesn't really matter.
	WinnerID uint `json:"winnerID" form:"winnerID" query:"winnerID"`
	LoserID  uint `json:"loserID" form:"loserID" query:"loserID"`
}

func (h *Handler) VoteHandler(c echo.Context) error {
	// TODO: validate request by sending a hashcode or something to prevent spamming
	//       vote requests without interacting with the page.
	req := new(VoteRequest)
	if err := c.Bind(req); err != nil {
		c.Logger().Warn("Vote request parsing error: ", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide valid IDs.")
	}
	c.Logger().Info("Received vote request: ", req)

	go h.processVoteRequest(c, req.WinnerID, req.LoserID) // asynchronous to avoid blocking
	return c.HTML(http.StatusAccepted, "Vote accepted")   // accepted but may not be processed yet (could still be rejected)
}

func (h *Handler) processVoteRequest(c echo.Context, winnerID uint, loserID uint) {
	winner, err := h.scpCache.GetByID(winnerID)
	if err != nil {
		c.Logger().Error(fmt.Sprintf("Error finding SCP id: %d ", winnerID), err)
		return
	}
	loser, err := h.scpCache.GetByID(loserID)
	if err != nil {
		c.Logger().Error(fmt.Sprintf("Error finding SCP id: %d ", loserID), err)
		return
	}
	if winner == nil {
		c.Logger().Warn(fmt.Sprintf("Could not find SCP id: %d", winnerID))
		return
	}
	if loser == nil {
		c.Logger().Warn(fmt.Sprintf("Could not find SCP id: %d", loserID))
		return
	}

	// Calculate new Elo ratings
	K := 20.0
	Pwinner := eloExpectedProbability(winner.Rating, loser.Rating)
	Ploser := eloExpectedProbability(loser.Rating, winner.Rating)
	winnerDiff := K * (1.0 - Pwinner)
	loserDiff := K * (0.0 - Ploser)
	// fmt.Printf("Pwinner: %.4f\n", Pwinner)
	// fmt.Printf("Ploser: %.4f\n", Ploser)
	// fmt.Printf("winnerRating before: %.2f\n", winner.Rating)
	// fmt.Printf("loserRating before: %.2f\n", loser.Rating)
	h.updateEloRating(winner, winnerDiff, 1, 0)
	h.updateEloRating(loser, loserDiff, 0, 1)
	// fmt.Printf("winnerRating after: %.2f\n", winner.Rating)
	// fmt.Printf("loserRating after: %.2f\n", loser.Rating)

	err = h.scpCache.Update(winner, loser)
	if err != nil {
		c.Logger().Error("Error during update: ", err)
	}
}

func (h *Handler) updateEloRating(scp *model.SCP, ratingDiff float64, winDiff uint64, lossDiff uint64) {
	// Use a fine-grained lock per SCP to prevent lost updates.
	if lock, ok := h.scpLock[scp.ID]; ok {
		lock.Lock()
		defer lock.Unlock()
	} else {
		lock = &sync.Mutex{}
		h.scpLock[scp.ID] = lock
		lock.Lock()
		defer lock.Unlock()
	}
	scp.Rating += ratingDiff
	scp.Wins += winDiff
	scp.Losses += lossDiff
}

func eloExpectedProbability(rating1 float64, rating2 float64) float64 {
	return 1.0 / (1.0 + math.Pow(10.0, (rating2-rating1)/400.0))
}
