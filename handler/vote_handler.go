package handler

import (
	"fmt"
	"math"
	"net/http"

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
		msg := fmt.Sprintf("Error finding SCP id: %d ", winnerID)
		c.Logger().Error(msg, err)
		return
	}
	loser, err := h.scpCache.GetByID(loserID)
	if err != nil {
		msg := fmt.Sprintf("Error finding SCP id: %d ", loserID)
		c.Logger().Error(msg, err)
		return
	}
	if winner == nil {
		msg := fmt.Sprintf("Could not find SCP id: %d", winnerID)
		c.Logger().Warn(msg)
		return
	}
	if loser == nil {
		msg := fmt.Sprintf("Could not find SCP id: %d", loserID)
		c.Logger().Warn(msg)
		return
	}

	winner.Wins++
	loser.Losses++
	// TODO: Elo rating
	K := 42.0
	Pwinner := eloExpectedProbability(winner.Rating, loser.Rating)
	Ploser := eloExpectedProbability(loser.Rating, winner.Rating)
	fmt.Printf("Pwinner: %.4f\n", Pwinner)
	fmt.Printf("Ploser: %.4f\n", Ploser)
	fmt.Printf("winnerRating before: %.2f\n", winner.Rating)
	fmt.Printf("loserRating before: %.2f\n", loser.Rating)
	winner.Rating = winner.Rating + K*(1.0-Pwinner)
	loser.Rating = loser.Rating + K*(0.0-Ploser)
	fmt.Printf("winnerRating after: %.2f\n", winner.Rating)
	fmt.Printf("loserRating after: %.2f\n", loser.Rating)

	// TODO: use a lock/atomic increments to ensure there are no lost updates.
	// Just calculate the difference and pass it to a goroutine or something to
	// increment/decrement the current values.
	// Yes this can bias the Elo ratings since if the base values between
	// calculating the rating difference to add and updating it, theoretically
	// the difference calculated would need change to be be more fair.
	// However, it's better than missing a vote entirely.
	h.scpCache.Update(winner, loser)
}

func eloExpectedProbability(rating1 float64, rating2 float64) float64 {
	return 1.0 / (1.0 + math.Pow(10.0, (rating2-rating1)/400.0))
}
