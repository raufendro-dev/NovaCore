package auth

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name         string `json:"name" gorm:"size:120;not null"`
	Email        string `json:"email" gorm:"size:180;uniqueIndex;not null"`
	PasswordHash string `json:"-" gorm:"not null"`
	Role         string `json:"role" gorm:"size:40;not null;default:user"`
}

func (u User) Roles() []string {
	if u.Role == "" {
		return []string{"user"}
	}
	return []string{u.Role}
}
