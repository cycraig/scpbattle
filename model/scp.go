package model

import (
	"github.com/jinzhu/gorm"
)

type SCP struct {
	gorm.Model
	Name        string `gorm:"unique_index;not null"`
	Description string
	Image       string
	Link        string
	Rating      float64
	Wins        uint64
	Losses      uint64
}

func NewSCP(name string, desc string, image string, link string) *SCP {
	return &SCP{
		Name:        name,
		Description: desc,
		Image:       image,
		Link:        link,
		Rating:      1000.0,
		Wins:        0,
		Losses:      0,
	}
}
