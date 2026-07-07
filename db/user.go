package db

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"time"
)

// ──────────────────────────────────────────────────────────
//  User model & CRUD
// ──────────────────────────────────────────────────────────

type User struct {
	ID           int
	Email        string
	Password     string
	AuthProvider string
	ProviderID   string
	CreatedAt    time.Time
}

func CreateUser(email, hashedPassword string) (int, error) {
	var id int
	err := DB.QueryRow(
		`INSERT INTO users (email, password, auth_provider)
		 VALUES ($1, $2, 'email') RETURNING id`,
		email, hashedPassword,
	).Scan(&id)
	return id, err
}

func GetUserByEmail(email string) (*User, error) {
	u := &User{}
	err := DB.QueryRow(
		`SELECT id, email, COALESCE(password,''), COALESCE(auth_provider,'email'),
		        COALESCE(provider_id,''), created_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.Password, &u.AuthProvider, &u.ProviderID, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func GetUserByID(id int) (*User, error) {
	u := &User{}
	err := DB.QueryRow(
		`SELECT id, email, COALESCE(auth_provider,'email'), created_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.AuthProvider, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// GetOrCreateOAuthUser finds or creates a user authenticated via OAuth.
func GetOrCreateOAuthUser(email, provider, providerID string) (*User, error) {
	// Try to find by provider + provider_id first
	u := &User{}
	err := DB.QueryRow(
		`SELECT id, email, COALESCE(auth_provider,''), created_at
		 FROM users WHERE auth_provider = $1 AND provider_id = $2`,
		provider, providerID,
	).Scan(&u.ID, &u.Email, &u.AuthProvider, &u.CreatedAt)
	if err == nil {
		return u, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Not found by provider_id — try by email
	existing, err := GetUserByEmail(email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		// Link OAuth to existing email account
		_, err = DB.Exec(
			`UPDATE users SET auth_provider = $1, provider_id = $2 WHERE id = $3`,
			provider, providerID, existing.ID,
		)
		existing.AuthProvider = provider
		return existing, err
	}

	// Create new user (no password — OAuth only)
	var id int
	err = DB.QueryRow(
		`INSERT INTO users (email, auth_provider, provider_id)
		 VALUES ($1, $2, $3) RETURNING id`,
		email, provider, providerID,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &User{ID: id, Email: email, AuthProvider: provider, CreatedAt: time.Now()}, nil
}

// ──────────────────────────────────────────────────────────
//  OTP helpers
// ──────────────────────────────────────────────────────────

func GenerateOTP() string {
	b := make([]byte, 3)
	rand.Read(b)
	return fmt.Sprintf("%06d", int(b[0])<<16|int(b[1])<<8|int(b[2])%1000000)
}

func StoreOTP(email, code string) error {
	// Invalidate old codes for this email
	_, _ = DB.Exec(`UPDATE otp_codes SET used = TRUE WHERE email = $1 AND used = FALSE`, email)

	_, err := DB.Exec(
		`INSERT INTO otp_codes (email, code, expires_at)
		 VALUES ($1, $2, $3)`,
		email, code, time.Now().Add(10*time.Minute),
	)
	return err
}

func VerifyOTP(email, code string) (bool, error) {
	var id int
	err := DB.QueryRow(
		`SELECT id FROM otp_codes
		 WHERE email = $1 AND code = $2 AND used = FALSE AND expires_at > NOW()
		 LIMIT 1`,
		email, code,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// Mark as used
	_, _ = DB.Exec(`UPDATE otp_codes SET used = TRUE WHERE id = $1`, id)
	return true, nil
}
