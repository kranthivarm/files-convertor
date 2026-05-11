package controllers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

			
	"i-lov-pdf/db"
	"i-lov-pdf/middleware"


)


type registerReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email required and password must be ≥ 6 characters"})
		return
	}

	existing, err := db.GetUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not hash password"})
		return
	}

	userID, err := db.CreateUser(req.Email, string(hashed))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create user"})
		return
	}

	token, err := middleware.Sign(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not sign token"})
		return
	}

	setTokenCookie(c, token)
	c.JSON(http.StatusCreated, gin.H{
		"message": "account created",
		"token":   token,
		"user":    gin.H{"id": userID, "email": req.Email},
	})
}


func Login(c *gin.Context) {
	var req registerReq // same shape
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	user, err := db.GetUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	token, err := middleware.Sign(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not sign token"})
		return
	}

	setTokenCookie(c, token)
	c.JSON(http.StatusOK, gin.H{
		"message": "logged in",
		"token":   token,
		"user":    gin.H{"id": user.ID, "email": user.Email},
	})
}


func Logout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}


func Me(c *gin.Context) {
	userID := middleware.GetUserID(c)
	user, err := db.GetUserByID(userID)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":         user.ID,
		"email":      user.Email,
		"created_at": user.CreatedAt,
	})
}


func History(c *gin.Context) {
	userID := middleware.GetUserID(c)
	activities, err := db.GetHistory(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch history"})
		return
	}
	if activities == nil {
		activities = []db.Activity{} // return [] not null
	}
	c.JSON(http.StatusOK, gin.H{"history": activities})
}


func setTokenCookie(c *gin.Context, token string) {
	c.SetCookie(
		"token",
		token,
		int((7 * 24 * time.Hour).Seconds()),
		"/",
		"",
		false,
		true, 
	)
}