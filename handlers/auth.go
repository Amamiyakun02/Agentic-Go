package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"AgenticGo/models"
	"AgenticGo/services"
	"AgenticGo/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// @Summary Register a new user
// @Description Create a new customer account
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.UserRegisterRequest true "User Registration Info"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/auth/register [post]
func RegisterUser(c *gin.Context) {
	var req models.UserRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Data tidak valid: " + err.Error()})
		return
	}

	emailClean := strings.ToLower(strings.TrimSpace(req.Email))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check existing
	var existingUser models.User
	err := services.UsersCol.FindOne(ctx, bson.M{"email": emailClean}).Decode(&existingUser)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Email sudah terdaftar. Silakan masuk menggunakan akun Anda."})
		return
	}

	// Hash password
	hashed, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Gagal memproses password."})
		return
	}

	var phone *string
	if req.Phone != "" {
		p := strings.TrimSpace(req.Phone)
		phone = &p
	}

	newUser := models.User{
		ID:           bson.NewObjectID(),
		Name:         strings.TrimSpace(req.Name),
		Email:        emailClean,
		Phone:        phone,
		PasswordHash: hashed,
		Role:         "customer",
		AvatarURL:    "https://api.dicebear.com/7.x/bottts/svg?seed=" + strings.TrimSpace(req.Name),
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	res, err := services.UsersCol.InsertOne(ctx, newUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Gagal menyimpan akun."})
		return
	}

	tokenData := map[string]interface{}{
		"id":    res.InsertedID.(bson.ObjectID).Hex(),
		"email": newUser.Email,
		"role":  newUser.Role,
	}

	accessToken, _ := utils.CreateAccessToken(tokenData)

	c.JSON(http.StatusCreated, gin.H{
		"status":       "success",
		"message":      "Registrasi akun member berhasil!",
		"access_token": accessToken,
		"user": gin.H{
			"id":         res.InsertedID.(bson.ObjectID).Hex(),
			"name":       newUser.Name,
			"email":      newUser.Email,
			"phone":      newUser.Phone,
			"role":       newUser.Role,
			"avatar_url": newUser.AvatarURL,
		},
	})
}

// @Summary Login user
// @Description Authenticate a user and return an access token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.UserLoginRequest true "User Login Info"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/auth/login [post]
func LoginUser(c *gin.Context) {
	var req models.UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Data tidak valid: " + err.Error()})
		return
	}

	emailClean := strings.ToLower(strings.TrimSpace(req.Email))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err := services.UsersCol.FindOne(ctx, bson.M{"email": emailClean}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusUnauthorized, gin.H{"detail": "Email atau kata sandi Anda salah."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Gagal mencari data pengguna."})
		return
	}

	// Verify Password
	isValid := utils.VerifyPassword(req.Password, user.PasswordHash)
	if !isValid {
		c.JSON(http.StatusUnauthorized, gin.H{"detail": "Email atau kata sandi Anda salah."})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"detail": "Akun Anda dinonaktifkan oleh administrator."})
		return
	}

	// Handle ObjectId conversion safely
	var userIdStr string
	if oid, ok := user.ID.(bson.ObjectID); ok {
		userIdStr = oid.Hex()
	} else if idStr, ok := user.ID.(string); ok {
		userIdStr = idStr
	}

	tokenData := map[string]interface{}{
		"id":    userIdStr,
		"email": user.Email,
		"role":  user.Role,
	}

	accessToken, _ := utils.CreateAccessToken(tokenData)

	c.JSON(http.StatusOK, gin.H{
		"status":       "success",
		"message":      "Login berhasil! Selamat datang kembali.",
		"access_token": accessToken,
		"user": gin.H{
			"id":         userIdStr,
			"name":       user.Name,
			"email":      user.Email,
			"phone":      user.Phone,
			"role":       user.Role,
			"avatar_url": user.AvatarURL,
		},
	})
}
