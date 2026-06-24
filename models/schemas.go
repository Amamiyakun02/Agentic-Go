package models

import "time"

// UserRegisterRequest merepresentasikan body JSON untuk registrasi
type UserRegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Phone    string `json:"phone"`
	Password string `json:"password" binding:"required,min=6"`
}

// UserLoginRequest merepresentasikan body JSON untuk login
type UserLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// GoogleLoginRequest merepresentasikan credential ID token Firebase dari client
type GoogleLoginRequest struct {
	Credential string `json:"credential" binding:"required"`
}

// User merepresentasikan dokumen koleksi users di MongoDB
type User struct {
	ID           interface{} `bson:"_id,omitempty" json:"id"`
	Name         string      `bson:"name" json:"name"`
	Email        string      `bson:"email" json:"email"`
	Phone        *string     `bson:"phone" json:"phone"`
	PasswordHash string      `bson:"password_hash" json:"-"` // Hidden from JSON response
	Role         string      `bson:"role" json:"role"`
	AvatarURL    string      `bson:"avatar_url" json:"avatar_url"`
	IsActive     bool        `bson:"is_active" json:"is_active"`
	CreatedAt    time.Time   `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time   `bson:"updated_at" json:"updated_at"`
}

// SyncDeviceAppsRequest merepresentasikan data JSON inventory aplikasi
type SyncDeviceAppsRequest struct {
	UserID    string                   `json:"user_id" binding:"required"`
	DeviceID  string                   `json:"device_id" binding:"required"`
	Apps      []map[string]interface{} `json:"apps" binding:"required"`
	Timestamp string                   `json:"timestamp"`
}
