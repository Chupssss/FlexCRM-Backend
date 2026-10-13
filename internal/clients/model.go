package clients

import (
	"time"

	"github.com/google/uuid"
)

type Client struct {
	ID   uuid.UUID `json:"id"`
	Type string    `json:"clientType"`

	FirstName   *string `json:"firstName,omitempty"`
	LastName    *string `json:"lastName,omitempty"`
	CompanyName *string `json:"companyName,omitempty"`

	Email *string `json:"email,omitempty"`
	Phone *string `json:"phone,omitempty"`

	PrimaryContact *Contact `json:"primaryContact,omitempty"`
	Responsible    *User    `json:"responsible,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Contact struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  *string   `json:"lastName,omitempty"`
	Email     *string   `json:"email,omitempty"`
	Phone     *string   `json:"phone,omitempty"`
}

type User struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  *string   `json:"lastName,omitempty"`
}
