package clients

import (
	"github.com/google/uuid"
	"time"
)

type Client struct {
	ID              uuid.UUID `json:"id"`
	Type            string    `json:"client_type"`
	FirstName       *string   `json:"first_name"`
	LastName        *string   `json:"last_name"`
	CompanyName     *string   `json:"company_name"`
	Email           *string   `json:"email"`
	Phone           *string   `json:"phone"`
	ResponsibleName *string   `json:"responsible_user_name"`
}

type ClientDetail struct {
	Client
	MiddleName    *string      `json:"middle_name"`
	ResponsibleID *uuid.UUID   `json:"responsible_user_id"`
	Description   *string      `json:"description"`
	LastComment   *LastComment `json:"last_comment"`
	Tags          []string     `json:"tags"`
	Contacts      []Contact    `json:"contacts"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

type ClientInput struct {
	Type          string     `json:"client_type"`
	FirstName     *string    `json:"first_name"`
	LastName      *string    `json:"last_name"`
	MiddleName    *string    `json:"middle_name"`
	CompanyName   *string    `json:"company_name"`
	Email         *string    `json:"email"`
	Phone         *string    `json:"phone"`
	ResponsibleID *uuid.UUID `json:"responsible_user_id"`
	Description   *string    `json:"description"`
}

type Contact struct {
	ID uuid.UUID `json:"id"`
	ContactInput
}

type ContactInput struct {
	FirstName string  `json:"first_name"`
	LastName  *string `json:"last_name"`
	Position  *string `json:"position"`
	Email     *string `json:"email"`
	Phone     *string `json:"phone"`
	IsPrimary bool    `json:"is_primary"`
}

type User struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  *string   `json:"last_name"`
}

type LastComment struct {
	Text       string    `json:"text"`
	AuthorName string    `json:"author_name"`
	CreatedAt  time.Time `json:"created_at"`
}

type Comment struct {
	ID       uuid.UUID `json:"id"`
	AuthorID uuid.UUID `json:"author_id"`
	LastComment
	UpdatedAt time.Time `json:"updated_at"`
}

type Tag struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type Filter struct {
	Search        string
	Type          string
	ResponsibleID *uuid.UUID
}

// Profile — ответ карточки клиента, строго по ClientDetail из Swagger.
// Поля записи БД и признак основного контакта в этот контракт не входят.
type Profile struct {
	Client
	Description *string          `json:"description"`
	LastComment *LastComment     `json:"last_comment"`
	Tags        []string         `json:"tags"`
	Contacts    []ProfileContact `json:"contacts"`
}

type ProfileContact struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  *string   `json:"last_name"`
	Phone     *string   `json:"phone"`
	Email     *string   `json:"email"`
	Position  *string   `json:"position"`
}

func profileFromDetail(detail *ClientDetail) Profile {
	profile := Profile{Client: detail.Client, Description: detail.Description, LastComment: detail.LastComment,
		Tags: make([]string, 0, len(detail.Tags)), Contacts: make([]ProfileContact, 0, len(detail.Contacts))}
	profile.Tags = append(profile.Tags, detail.Tags...)
	if detail.Type == "COMPANY" {
		for _, contact := range detail.Contacts {
			profile.Contacts = append(profile.Contacts, ProfileContact{ID: contact.ID, FirstName: contact.FirstName,
				LastName: contact.LastName, Phone: contact.Phone, Email: contact.Email, Position: contact.Position})
		}
	}
	return profile
}
