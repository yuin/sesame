package domain

import "time"

type User struct {
	ID        string
	Name      string
	UpdatedAt time.Time
}
