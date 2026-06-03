// Package model holds the domain entities (User, Donation, Post, …) shared
// across the repository, service, and handler layers.
package model

// User is the core user account record.
type User struct {
	ID           int64  `json:"id"`
	FullName     string `json:"full_name"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
}

// RegisterRequest is the payload for account creation.
type RegisterRequest struct {
	FullName        string `json:"full_name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
	Username        string `json:"username"`
	Bio             string `json:"bio"`
	CategoryID      string `json:"category_id"`
}
