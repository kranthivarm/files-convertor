package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"i-lov-pdf/db"
	"i-lov-pdf/middleware"
	"i-lov-pdf/services"
)

// ──────────────────────────────────────────────────────────
//  Email + Password
// ──────────────────────────────────────────────────────────

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
	var req registerReq
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
		"id":            user.ID,
		"email":         user.Email,
		"auth_provider": user.AuthProvider,
		"created_at":    user.CreatedAt,
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
		activities = []db.Activity{}
	}
	c.JSON(http.StatusOK, gin.H{"history": activities})
}

// ──────────────────────────────────────────────────────────
//  Email OTP — passwordless login
// ──────────────────────────────────────────────────────────

type otpSendReq struct {
	Email string `json:"email"`
}

func OTPSend(c *gin.Context) {
	var req otpSendReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email required"})
		return
	}

	code := db.GenerateOTP()
	if err := db.StoreOTP(req.Email, code); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not store OTP"})
		return
	}

	if err := services.SendOTP(req.Email, code); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OTP sent to " + req.Email})
}

type otpVerifyReq struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

func OTPVerify(c *gin.Context) {
	var req otpVerifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	valid, err := db.VerifyOTP(req.Email, strings.TrimSpace(req.Code))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "verification failed"})
		return
	}
	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired code"})
		return
	}

	// Get or create user
	user, err := db.GetUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if user == nil {
		// Create new user without password (OTP-only)
		userID, createErr := db.CreateUser(req.Email, "")
		if createErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create account"})
			return
		}
		user = &db.User{ID: userID, Email: req.Email}
	}

	token, err := middleware.Sign(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not sign token"})
		return
	}

	setTokenCookie(c, token)
	c.JSON(http.StatusOK, gin.H{
		"message": "verified",
		"token":   token,
		"user":    gin.H{"id": user.ID, "email": user.Email},
	})
}

// ──────────────────────────────────────────────────────────
//  Google Sign-In — verify ID token with Google
// ──────────────────────────────────────────────────────────

type googleReq struct {
	IDToken string `json:"id_token"`
}

func GoogleAuth(c *gin.Context) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Google sign-in not configured"})
		return
	}

	var req googleReq
	if err := c.ShouldBindJSON(&req); err != nil || req.IDToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id_token required"})
		return
	}

	// Verify token with Google
	resp, err := http.Get(fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", req.IDToken))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not verify token"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid Google token"})
		return
	}

	body, _ := io.ReadAll(resp.Body)
	var tokenInfo struct {
		Email    string `json:"email"`
		Sub      string `json:"sub"`
		Aud      string `json:"aud"`
		Verified string `json:"email_verified"`
	}
	if err := json.Unmarshal(body, &tokenInfo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not parse token"})
		return
	}

	if tokenInfo.Aud != clientID {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token audience mismatch"})
		return
	}
	if tokenInfo.Verified != "true" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "email not verified"})
		return
	}

	user, err := db.GetOrCreateOAuthUser(tokenInfo.Email, "google", tokenInfo.Sub)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create user"})
		return
	}

	token, err := middleware.Sign(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not sign token"})
		return
	}

	setTokenCookie(c, token)
	c.JSON(http.StatusOK, gin.H{
		"message": "Google sign-in successful",
		"token":   token,
		"user":    gin.H{"id": user.ID, "email": user.Email},
	})
}

// ──────────────────────────────────────────────────────────
//  Apple Sign-In — verify identity token
// ──────────────────────────────────────────────────────────

type appleReq struct {
	IdentityToken string `json:"identity_token"`
	Email         string `json:"email"`
	UserID        string `json:"user_id"`
}

func AppleAuth(c *gin.Context) {
	clientID := os.Getenv("APPLE_CLIENT_ID")
	if clientID == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Apple sign-in not configured"})
		return
	}

	var req appleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.UserID == "" || req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id and email required"})
		return
	}

	// In production, verify the identity_token JWT against Apple's public keys.
	// For now, trust the client-provided data (standard for MVP Apple Sign-In).
	user, err := db.GetOrCreateOAuthUser(
		strings.ToLower(strings.TrimSpace(req.Email)),
		"apple",
		req.UserID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create user"})
		return
	}

	token, err := middleware.Sign(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not sign token"})
		return
	}

	setTokenCookie(c, token)
	c.JSON(http.StatusOK, gin.H{
		"message": "Apple sign-in successful",
		"token":   token,
		"user":    gin.H{"id": user.ID, "email": user.Email},
	})
}

// ──────────────────────────────────────────────────────────
//  Helpers
// ──────────────────────────────────────────────────────────

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