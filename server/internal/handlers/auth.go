package handlers

import (
	"net/http"

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

	token, err := h.jwt.GenerateToken(user.ID, user.Email)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to generate token", err)
		return
	}

	loginResponse := models.LoginResponse{
		Token: token,
		User:  user,
	}

	response.Success(c, http.StatusCreated, "User created successfully", loginResponse)
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

	token, err := h.jwt.GenerateToken(user.ID, user.Email)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to generate token", err)
		return
	}

	loginResponse := models.LoginResponse{
		Token: token,
		User:  user,
	}

	response.Success(c, http.StatusOK, "Login successful", loginResponse)
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
