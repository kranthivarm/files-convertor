package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// DB is the shared database handle. Call Connect() once at startup.
var DB *sql.DB

// Connect opens the PostgreSQL connection using DATABASE_URL env var.
// Falls back to a sensible default for local dev.
func Connect() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/pdfforge?sslmode=disable"
	}

	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("db open: %w", err)
	}

	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(5 * time.Minute)

	if err = DB.Ping(); err != nil {
		return fmt.Errorf("db ping: %w", err)
	}

	log.Println("✅ PostgreSQL connected")
	return nil
}

// Migrate creates the tables if they do not already exist.
func Migrate() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id         SERIAL PRIMARY KEY,
			email      TEXT UNIQUE NOT NULL,
			password   TEXT NOT NULL,           -- bcrypt hash
			created_at TIMESTAMPTZ DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS activity_log (
			id         SERIAL PRIMARY KEY,
			user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			operation  TEXT NOT NULL,           -- e.g. "merge", "compress"
			filename   TEXT NOT NULL,           -- original uploaded filename
			created_at TIMESTAMPTZ DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_activity_user ON activity_log(user_id);
	`)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	log.Println("✅ DB schema ready")
	return nil
}

// ── User ──────────────────────────────────────────────────────────────────────

type User struct {
	ID        int
	Email     string
	Password  string
	CreatedAt time.Time
}

func CreateUser(email, hashedPassword string) (int, error) {
	var id int
	err := DB.QueryRow(
		`INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id`,
		email, hashedPassword,
	).Scan(&id)
	return id, err
}

func GetUserByEmail(email string) (*User, error) {
	u := &User{}
	err := DB.QueryRow(
		`SELECT id, email, password, created_at FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.Password, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func GetUserByID(id int) (*User, error) {
	u := &User{}
	err := DB.QueryRow(
		`SELECT id, email, created_at FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// ── Activity Log ──────────────────────────────────────────────────────────────

type Activity struct {
	ID        int       `json:"id"`
	Operation string    `json:"operation"`
	Filename  string    `json:"filename"`
	CreatedAt time.Time `json:"created_at"`
}

// LogActivity inserts one row. Silently skips if userID <= 0 (guest).
func LogActivity(userID int, operation, filename string) {
	if userID <= 0 {
		return
	}
	_, err := DB.Exec(
		`INSERT INTO activity_log (user_id, operation, filename) VALUES ($1, $2, $3)`,
		userID, operation, filename,
	)
	if err != nil {
		log.Printf("activity log error: %v", err)
	}
}

// GetHistory returns the last 50 activities for the given user.
func GetHistory(userID int) ([]Activity, error) {
	rows, err := DB.Query(
		`SELECT id, operation, filename, created_at
		 FROM activity_log
		 WHERE user_id = $1
		 ORDER BY created_at DESC
		 LIMIT 50`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Activity
	for rows.Next() {
		var a Activity
		if err := rows.Scan(&a.ID, &a.Operation, &a.Filename, &a.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}