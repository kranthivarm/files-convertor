package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

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

	log.Println("PostgreSQL connected")
	return nil
}

func Migrate() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id            SERIAL PRIMARY KEY,
			email         TEXT UNIQUE NOT NULL,
			password      TEXT,
			auth_provider TEXT DEFAULT 'email',
			provider_id   TEXT,
			created_at    TIMESTAMPTZ DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS activity_log (
			id         SERIAL PRIMARY KEY,
			user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			operation  TEXT NOT NULL,
			filename   TEXT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_activity_user ON activity_log(user_id);

		CREATE TABLE IF NOT EXISTS otp_codes (
			id         SERIAL PRIMARY KEY,
			email      TEXT NOT NULL,
			code       TEXT NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			used       BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_otp_email ON otp_codes(email);
	`)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	log.Println("DB schema ready")
	return nil
}
