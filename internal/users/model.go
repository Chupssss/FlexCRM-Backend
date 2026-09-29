package users

import (
	"time"
	"uuid"

	"github.com/google/uuid"
)

type User struct {
	ID            uuid.UUID
	CompanyID     uuid.UUID
	Email         string
	Password_Hash string
	First_Name    string
	Last_Name     *string
	Role          string
	Is_active     bool
	Created_at    time.Time
	Updated_at    time.Time
}
