package models

import (
	"gorm.io/gorm"
)

type Document struct {
	gorm.Model
	Url        string     `gorm:"unique;varchar(2048);not null"`
	Words      []WordFreq `gorm:"not null;foreignKey:DocumentID"`
	WordsCount uint       `gorm:"not null"`
}
