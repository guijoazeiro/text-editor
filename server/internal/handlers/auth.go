package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/auth"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
	"github.com/guijoazeiro/text-editor/tree/main/server/pkg/response"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db  *gorm.DB
	jwt *auth.JWT
}

func NewAuthHandler(db *gorm.DB, jwt *auth.JWT) *AuthHandler {
	return &AuthHandler{db: db, jwt: jwt}
}

func (h *AuthHandler) Signup(c *gin.Context) {
	var req models.SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	var existingUser models.User
	if err := h.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		response.Error(c, http.StatusConflict, "Email already registered", nil)
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to hash password", err)
		return
	}

	user := models.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: passwordHash,
	}

	if err := h.db.Create(&user).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create user", err)
		return
	}

	accessToken, err := h.jwt.GenerateToken(user.ID, user.Email)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to generate token", err)
		return
	}

	refreshToken, err := h.jwt.GenerateRefreshToken(user.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to generate refresh token", err)
		return
	}

	exp := time.Now().Add(30 * 24 * time.Hour)
	h.db.Model(&user).Updates(map[string]interface{}{
		"refresh_token_hash": auth.HashRefreshToken(refreshToken),
		"refresh_token_exp":  exp,
	})

	setRefreshCookie(c, refreshToken)

	response.Success(c, http.StatusCreated, "User created successfully", models.LoginResponse{
		Token: accessToken,
		User:  user,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		response.Error(c, http.StatusUnauthorized, "Invalid credentials", nil)
		return
	}

	if !auth.CheckPasswordHash(req.Password, user.PasswordHash) {
		response.Error(c, http.StatusUnauthorized, "Invalid credentials", nil)
		return
	}

	accessToken, err := h.jwt.GenerateToken(user.ID, user.Email)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to generate token", err)
		return
	}

	refreshToken, err := h.jwt.GenerateRefreshToken(user.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to generate refresh token", err)
		return
	}

	exp := time.Now().Add(30 * 24 * time.Hour)
	h.db.Model(&user).Updates(map[string]interface{}{
		"refresh_token_hash": auth.HashRefreshToken(refreshToken),
		"refresh_token_exp":  exp,
	})

	setRefreshCookie(c, refreshToken)

	response.Success(c, http.StatusOK, "Login successful", models.LoginResponse{
		Token: accessToken,
		User:  user,
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "No refresh token", nil)
		return
	}

	userID, err := h.jwt.ValidateRefreshToken(refreshToken)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Invalid refresh token", nil)
		return
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		response.Error(c, http.StatusUnauthorized, "User not found", nil)
		return
	}

	if auth.HashRefreshToken(refreshToken) != user.RefreshTokenHash ||
		user.RefreshTokenExp == nil ||
		time.Now().After(*user.RefreshTokenExp) {
		response.Error(c, http.StatusUnauthorized, "Refresh token expired or revoked", nil)
		return
	}

	newAccessToken, err := h.jwt.GenerateToken(user.ID, user.Email)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to generate token", err)
		return
	}

	response.Success(c, http.StatusOK, "Token refreshed", gin.H{"token": newAccessToken})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	h.db.Model(&models.User{}).Where("id = ?", userID.(uuid.UUID)).Updates(map[string]interface{}{
		"refresh_token_hash": "",
		"refresh_token_exp":  nil,
	})

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	response.Success(c, http.StatusOK, "Logged out successfully", nil)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", userID.(uuid.UUID)).Error; err != nil {
		response.Error(c, http.StatusNotFound, "User not found", err)
		return
	}

	response.Success(c, http.StatusOK, "User retrieved successfully", user)
}

func (h *AuthHandler) UpdateMe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", userID.(uuid.UUID)).Error; err != nil {
		response.Error(c, http.StatusNotFound, "User not found", err)
		return
	}

	if err := h.db.Model(&user).Update("name", req.Name).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to update profile", err)
		return
	}

	response.Success(c, http.StatusOK, "Profile updated successfully", user)
}

func setRefreshCookie(c *gin.Context, token string) {
	maxAge := 30 * 24 * 60 * 60
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("refresh_token", token, maxAge, "/", "", false, true)
}
