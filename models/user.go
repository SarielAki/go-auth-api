package models

import "github.com/jinzhu/gorm"

type User struct {
	gorm.Model
	Name     string `json:"username" gorm:"UNIQUE"`
	Password string `json:"password"`
}
