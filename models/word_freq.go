package models

import "gorm.io/gorm"

type WordFreq struct {
	gorm.Model
	Word       string `gorm:"not null;varchar(2048)"`
	Count      uint   `gorm:"not null"`
	DocumentID uint   `gorm:"not null"`
}
