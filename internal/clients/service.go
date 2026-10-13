package clients

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("запись не найдена")
var ErrInvalid = errors.New("некорректные данные")
var ErrConflict = errors.New("клиент связан со сделками или задачами")

type Service struct{ repository *Repository }

func NewService(repository *Repository) *Service { return &Service{repository: repository} }

func (s *Service) GetAll(ctx context.Context, companyID uuid.UUID, filter Filter) ([]Client, error) {
	if filter.Type != "" && filter.Type != "PERSON" && filter.Type != "COMPANY" {
		return nil, ErrInvalid
	}
	return s.repository.GetAll(ctx, companyID, filter)
}

func (s *Service) GetByID(ctx context.Context, companyID, id uuid.UUID) (*ClientDetail, error) {
	return s.repository.GetByID(ctx, companyID, id)
}

func clean(value **string) {
	if *value == nil {
		return
	}
	text := strings.TrimSpace(**value)
	if text == "" {
		*value = nil
	} else {
		*value = &text
	}
}

func validText(value *string, max int) bool {
	return value == nil || utf8.RuneCountInString(*value) <= max
}

func validEmail(value *string) bool {
	if value == nil {
		return true
	}
	address, err := mail.ParseAddress(*value)
	return err == nil && address.Address == *value && utf8.RuneCountInString(*value) <= 255
}

func validateClient(input *ClientInput) error {
	input.Type = strings.ToUpper(strings.TrimSpace(input.Type))
	for _, value := range []**string{&input.FirstName, &input.LastName, &input.MiddleName, &input.CompanyName, &input.Email, &input.Phone, &input.Description} {
		clean(value)
	}
	if input.Type != "PERSON" && input.Type != "COMPANY" {
		return ErrInvalid
	}
	if input.FirstName == nil || input.LastName == nil {
		return ErrInvalid
	}
	if input.Type == "PERSON" && input.CompanyName != nil {
		return ErrInvalid
	}
	if input.Type == "COMPANY" && input.CompanyName == nil {
		return ErrInvalid
	}
	if !validText(input.FirstName, 100) || !validText(input.LastName, 100) || !validText(input.MiddleName, 100) || !validText(input.CompanyName, 255) || !validText(input.Phone, 50) || !validText(input.Description, 10000) || !validEmail(input.Email) {
		return ErrInvalid
	}
	if input.ResponsibleID != nil && *input.ResponsibleID == uuid.Nil {
		return ErrInvalid
	}
	return nil
}

func (s *Service) Create(ctx context.Context, companyID uuid.UUID, input ClientInput) (*ClientDetail, error) {
	if err := validateClient(&input); err != nil {
		return nil, err
	}
	id, err := s.repository.Save(ctx, companyID, uuid.Nil, input)
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, companyID, id)
}

func (s *Service) Update(ctx context.Context, companyID, id uuid.UUID, input ClientInput) (*ClientDetail, error) {
	if err := validateClient(&input); err != nil {
		return nil, err
	}
	if _, err := s.repository.Save(ctx, companyID, id, input); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, companyID, id)
}

func (s *Service) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	return s.repository.Delete(ctx, companyID, id)
}

func (s *Service) GetUsers(ctx context.Context, companyID uuid.UUID) ([]User, error) {
	return s.repository.GetUsers(ctx, companyID)
}

func (s *Service) GetContacts(ctx context.Context, companyID, id uuid.UUID) ([]Contact, error) {
	return s.repository.GetContacts(ctx, companyID, id)
}

func validateContact(input *ContactInput) error {
	input.FirstName = strings.TrimSpace(input.FirstName)
	for _, value := range []**string{&input.LastName, &input.Position, &input.Email, &input.Phone} {
		clean(value)
	}
	if input.FirstName == "" || utf8.RuneCountInString(input.FirstName) > 100 || !validText(input.LastName, 100) || !validText(input.Position, 150) || !validText(input.Phone, 50) || !validEmail(input.Email) {
		return ErrInvalid
	}
	return nil
}

func (s *Service) SaveContact(ctx context.Context, companyID, clientID, id uuid.UUID, input ContactInput) (*Contact, error) {
	if err := validateContact(&input); err != nil {
		return nil, err
	}
	return s.repository.SaveContact(ctx, companyID, clientID, id, input)
}

func (s *Service) DeleteContact(ctx context.Context, companyID, clientID, id uuid.UUID) error {
	return s.repository.DeleteContact(ctx, companyID, clientID, id)
}

func (s *Service) GetComments(ctx context.Context, companyID, id uuid.UUID) ([]Comment, error) {
	return s.repository.GetComments(ctx, companyID, id)
}

func (s *Service) SaveComment(ctx context.Context, companyID, clientID, id uuid.UUID, text string) (*Comment, error) {
	text = strings.TrimSpace(text)
	if text == "" || utf8.RuneCountInString(text) > 10000 {
		return nil, ErrInvalid
	}
	return s.repository.SaveComment(ctx, companyID, clientID, id, text)
}

func (s *Service) DeleteComment(ctx context.Context, companyID, clientID, id uuid.UUID) error {
	return s.repository.DeleteComment(ctx, companyID, clientID, id)
}

func (s *Service) GetTags(ctx context.Context, companyID uuid.UUID) ([]Tag, error) {
	return s.repository.GetTags(ctx, companyID)
}

func (s *Service) GetClientTags(ctx context.Context, companyID, id uuid.UUID) ([]Tag, error) {
	return s.repository.GetClientTags(ctx, companyID, id)
}

func (s *Service) CreateTag(ctx context.Context, companyID uuid.UUID, name string) (*Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return nil, ErrInvalid
	}
	return s.repository.CreateTag(ctx, companyID, name)
}

func (s *Service) SetTags(ctx context.Context, companyID, id uuid.UUID, tags []uuid.UUID) error {
	if tags == nil || len(tags) > 100 {
		return ErrInvalid
	}
	unique := make([]uuid.UUID, 0, len(tags))
	seen := make(map[uuid.UUID]bool)
	for _, tag := range tags {
		if tag == uuid.Nil {
			return ErrInvalid
		}
		if !seen[tag] {
			unique = append(unique, tag)
			seen[tag] = true
		}
	}
	return s.repository.SetTags(ctx, companyID, id, unique)
}
