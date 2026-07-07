package db

import (
	"log"
	"time"
)

// ──────────────────────────────────────────────────────────
//  Activity model & CRUD
// ──────────────────────────────────────────────────────────

type Activity struct {
	ID        int       `json:"id"`
	Operation string    `json:"operation"`
	Filename  string    `json:"filename"`
	CreatedAt time.Time `json:"created_at"`
}

func LogActivity(userID int, operation, filename string) {
	if userID <= 0 || DB == nil {
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
