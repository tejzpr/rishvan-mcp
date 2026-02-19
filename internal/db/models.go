package db

import (
	"time"

	"gorm.io/gorm"
)

type Request struct {
	gorm.Model
	SourceName  string     `json:"source_name" gorm:"column:source_name;index;not null;default:''"`
	AppName     string     `json:"app_name" gorm:"index;not null"`
	Question    string     `json:"question" gorm:"type:text;not null"`
	Response    string     `json:"response" gorm:"type:text"`
	Status      string     `json:"status" gorm:"default:pending;not null;index"`
	RespondedAt *time.Time `json:"responded_at"`
}
