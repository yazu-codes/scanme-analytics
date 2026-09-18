package models

type User struct {
	ID              uint   `gorm:"primaryKey" json:"id"`
	Email           string `gorm:"uniqueIndex" json:"email"`
	Password        string `json:"-"` // Never expose password in JSON
	Name            string `json:"name"`
	Role            string `json:"role"`
	AssociatedMenus string `json:"menus"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type SignupRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
