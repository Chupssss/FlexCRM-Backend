package clients

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

const clientFields = `c.id, c.client_type,
 CASE WHEN c.client_type='COMPANY' THEN COALESCE(pc.first_name,c.first_name) ELSE c.first_name END,
 CASE WHEN c.client_type='COMPANY' THEN COALESCE(pc.last_name,c.last_name) ELSE c.last_name END,
 c.company_name, c.email, c.phone,
 CASE WHEN u.id IS NOT NULL THEN concat_ws(' ',u.first_name,u.last_name) END`

const clientJoins = ` FROM clients c
 LEFT JOIN users u ON u.id=c.responsible_user_id AND u.company_id=c.company_id
 LEFT JOIN LATERAL (
 SELECT first_name,last_name FROM contacts
 WHERE client_id=c.id AND is_primary=TRUE
 ORDER BY updated_at DESC,id LIMIT 1
 ) pc ON TRUE `

func (r *Repository) GetAll(ctx context.Context, companyID uuid.UUID, filter Filter) ([]Client, error) {
	rows, err := r.db.Query(ctx, `SELECT `+clientFields+clientJoins+`
 WHERE c.company_id=$1 AND ($2='' OR c.client_type=$2)
 AND ($3::uuid IS NULL OR c.responsible_user_id=$3)
 AND ($4='' OR concat_ws(' ',c.first_name,c.last_name,c.company_name,c.email,c.phone,pc.first_name,pc.last_name) ILIKE '%' || $4 || '%')
 ORDER BY c.created_at DESC,c.id`, companyID, filter.Type, filter.ResponsibleID, filter.Search)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	clients := make([]Client, 0)
	for rows.Next() {
		var c Client
		if err = rows.Scan(&c.ID, &c.Type, &c.FirstName, &c.LastName, &c.CompanyName, &c.Email, &c.Phone, &c.ResponsibleName); err != nil {
			return nil, err
		}
		clients = append(clients, c)
	}
	return clients, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, companyID, id uuid.UUID) (*ClientDetail, error) {
	var c ClientDetail
	err := r.db.QueryRow(ctx, `SELECT `+clientFields+`,c.middle_name,c.responsible_user_id,c.description,c.created_at,c.updated_at`+clientJoins+`WHERE c.company_id=$1 AND c.id=$2`, companyID, id).Scan(&c.ID, &c.Type, &c.FirstName, &c.LastName, &c.CompanyName, &c.Email, &c.Phone, &c.ResponsibleName, &c.MiddleName, &c.ResponsibleID, &c.Description, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.Tags = make([]string, 0)
	c.Contacts = make([]Contact, 0)
	rows, err := r.db.Query(ctx, `SELECT DISTINCT t.name FROM tags t JOIN client_tags ct ON ct.tag_id=t.id WHERE ct.client_id=$1 AND t.company_id=$2 ORDER BY t.name`, id, companyID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			rows.Close()
			return nil, err
		}
		c.Tags = append(c.Tags, name)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	var last LastComment
	err = r.db.QueryRow(ctx, `SELECT cm.text,concat_ws(' ',u.first_name,u.last_name),cm.created_at FROM comments cm JOIN users u ON u.id=cm.author_id AND u.company_id=cm.company_id WHERE cm.client_id=$1 AND cm.company_id=$2 ORDER BY cm.created_at DESC,cm.id DESC LIMIT 1`, id, companyID).Scan(&last.Text, &last.AuthorName, &last.CreatedAt)
	if err == nil {
		c.LastComment = &last
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if c.Type == "COMPANY" {
		contacts, err := r.GetContacts(ctx, companyID, id)
		if err != nil {
			return nil, err
		}
		for _, contact := range contacts {
			if !contact.IsPrimary {
				c.Contacts = append(c.Contacts, contact)
			}
		}
	}
	return &c, nil
}

func rollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = tx.Rollback(ctx)
}

func (r *Repository) clientTransaction(ctx context.Context, companyID, id uuid.UUID) (pgx.Tx, string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, "", err
	}
	var kind string
	err = tx.QueryRow(ctx, `SELECT client_type FROM clients WHERE company_id=$1 AND id=$2 FOR UPDATE`, companyID, id).Scan(&kind)
	if err != nil {
		rollback(tx)
		if errors.Is(err, pgx.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, "", err
	}
	return tx, kind, nil
}

func demoAuthor(ctx context.Context, tx pgx.Tx, companyID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `SELECT id FROM users WHERE company_id=$1 AND is_active=TRUE ORDER BY (role='ADMIN') DESC,created_at,id LIMIT 1`, companyID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("в тестовой компании нет активного пользователя")
	}
	return id, err
}

func (r *Repository) Save(ctx context.Context, companyID, id uuid.UUID, input ClientInput) (uuid.UUID, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer rollback(tx)
	var oldType string
	if id != uuid.Nil {
		err = tx.QueryRow(ctx, `SELECT client_type FROM clients WHERE company_id=$1 AND id=$2 FOR UPDATE`, companyID, id).Scan(&oldType)
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrNotFound
		}
		if err != nil {
			return uuid.Nil, err
		}
	}
	if input.ResponsibleID != nil {
		var exists bool
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND company_id=$2 AND is_active=TRUE)`, input.ResponsibleID, companyID).Scan(&exists)
		if err != nil {
			return uuid.Nil, err
		}
		if !exists {
			return uuid.Nil, ErrInvalid
		}
	}
	if id == uuid.Nil {
		author, err := demoAuthor(ctx, tx, companyID)
		if err != nil {
			return uuid.Nil, err
		}
		id = uuid.New()
		_, err = tx.Exec(ctx, `INSERT INTO clients (id,company_id,created_by,responsible_user_id,client_type,first_name,last_name,middle_name,company_name,email,phone,description,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NOW(),NOW())`, id, companyID, author, input.ResponsibleID, input.Type, input.FirstName, input.LastName, input.MiddleName, input.CompanyName, input.Email, input.Phone, input.Description)
		if err != nil {
			return uuid.Nil, err
		}
	} else {
		if input.Type == "PERSON" && oldType == "COMPANY" {
			var contacts bool
			if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM contacts WHERE client_id=$1)`, id).Scan(&contacts); err != nil {
				return uuid.Nil, err
			}
			if contacts {
				return uuid.Nil, ErrInvalid
			}
		}
		_, err = tx.Exec(ctx, `UPDATE clients SET responsible_user_id=$3,client_type=$4,first_name=$5,last_name=$6,middle_name=$7,company_name=$8,email=$9,phone=$10,description=$11,updated_at=NOW() WHERE company_id=$1 AND id=$2`, companyID, id, input.ResponsibleID, input.Type, input.FirstName, input.LastName, input.MiddleName, input.CompanyName, input.Email, input.Phone, input.Description)
		if err != nil {
			return uuid.Nil, err
		}
		_, err = tx.Exec(ctx, `UPDATE contacts SET first_name=$2,last_name=$3,updated_at=NOW() WHERE client_id=$1 AND is_primary=TRUE`, id, input.FirstName, input.LastName)
		if err != nil {
			return uuid.Nil, err
		}
	}
	return id, tx.Commit(ctx)
}

func (r *Repository) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	tx, _, err := r.clientTransaction(ctx, companyID, id)
	if err != nil {
		return err
	}
	defer rollback(tx)
	var linked bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM deals WHERE client_id=$1) OR EXISTS(SELECT 1 FROM tasks WHERE client_id=$1)`, id).Scan(&linked)
	if err != nil {
		return err
	}
	if linked {
		return ErrConflict
	}
	if _, err = tx.Exec(ctx, `DELETE FROM client_tags WHERE client_id=$1`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM contacts WHERE client_id=$1`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM comments WHERE client_id=$1 AND company_id=$2`, id, companyID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM activities WHERE entity_id=$1 AND company_id=$2 AND entity_type='CLIENT'`, id, companyID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM clients WHERE id=$1 AND company_id=$2`, id, companyID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) GetUsers(ctx context.Context, companyID uuid.UUID) ([]User, error) {
	rows, err := r.db.Query(ctx, `SELECT id,first_name,last_name FROM users WHERE company_id=$1 AND is_active=TRUE ORDER BY first_name,last_name,id`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := make([]User, 0)
	for rows.Next() {
		var u User
		if err = rows.Scan(&u.ID, &u.FirstName, &u.LastName); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *Repository) GetContacts(ctx context.Context, companyID, id uuid.UUID) ([]Contact, error) {
	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM clients WHERE company_id=$1 AND id=$2)`, companyID, id).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}
	rows, err := r.db.Query(ctx, `SELECT id,first_name,last_name,position,email,phone,is_primary FROM contacts WHERE client_id=$1 ORDER BY is_primary DESC,created_at,id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	contacts := make([]Contact, 0)
	for rows.Next() {
		var c Contact
		if err = rows.Scan(&c.ID, &c.FirstName, &c.LastName, &c.Position, &c.Email, &c.Phone, &c.IsPrimary); err != nil {
			return nil, err
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}

func (r *Repository) SaveContact(ctx context.Context, companyID, clientID, id uuid.UUID, input ContactInput) (*Contact, error) {
	tx, kind, err := r.clientTransaction(ctx, companyID, clientID)
	if err != nil {
		return nil, err
	}
	defer rollback(tx)
	if kind != "COMPANY" || (input.IsPrimary && input.LastName == nil) {
		return nil, ErrInvalid
	}
	if id != uuid.Nil {
		var exists bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM contacts WHERE id=$1 AND client_id=$2)`, id, clientID).Scan(&exists); err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrNotFound
		}
	}
	if input.IsPrimary {
		if _, err = tx.Exec(ctx, `UPDATE contacts SET is_primary=FALSE,updated_at=NOW() WHERE client_id=$1 AND is_primary=TRUE`, clientID); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(ctx, `UPDATE clients SET first_name=$2,last_name=$3,updated_at=NOW() WHERE id=$1`, clientID, input.FirstName, input.LastName); err != nil {
			return nil, err
		}
	}
	if id == uuid.Nil {
		id = uuid.New()
		_, err = tx.Exec(ctx, `INSERT INTO contacts (id,client_id,first_name,last_name,position,email,phone,is_primary,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW(),NOW())`, id, clientID, input.FirstName, input.LastName, input.Position, input.Email, input.Phone, input.IsPrimary)
	} else {
		_, err = tx.Exec(ctx, `UPDATE contacts SET first_name=$3,last_name=$4,position=$5,email=$6,phone=$7,is_primary=$8,updated_at=NOW() WHERE id=$1 AND client_id=$2`, id, clientID, input.FirstName, input.LastName, input.Position, input.Email, input.Phone, input.IsPrimary)
	}
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &Contact{ID: id, ContactInput: input}, nil
}

func (r *Repository) DeleteContact(ctx context.Context, companyID, clientID, id uuid.UUID) error {
	tx, _, err := r.clientTransaction(ctx, companyID, clientID)
	if err != nil {
		return err
	}
	defer rollback(tx)
	result, err := tx.Exec(ctx, `DELETE FROM contacts WHERE id=$1 AND client_id=$2`, id, clientID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

func (r *Repository) GetComments(ctx context.Context, companyID, id uuid.UUID) ([]Comment, error) {
	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM clients WHERE company_id=$1 AND id=$2)`, companyID, id).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}
	rows, err := r.db.Query(ctx, `SELECT cm.id,cm.author_id,cm.text,concat_ws(' ',u.first_name,u.last_name),cm.created_at,cm.updated_at FROM comments cm JOIN users u ON u.id=cm.author_id AND u.company_id=cm.company_id WHERE cm.company_id=$1 AND cm.client_id=$2 ORDER BY cm.created_at DESC,cm.id DESC`, companyID, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	comments := make([]Comment, 0)
	for rows.Next() {
		var c Comment
		if err = rows.Scan(&c.ID, &c.AuthorID, &c.Text, &c.AuthorName, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func (r *Repository) SaveComment(ctx context.Context, companyID, clientID, id uuid.UUID, text string) (*Comment, error) {
	tx, _, err := r.clientTransaction(ctx, companyID, clientID)
	if err != nil {
		return nil, err
	}
	defer rollback(tx)
	var c Comment
	if id == uuid.Nil {
		author, err := demoAuthor(ctx, tx, companyID)
		if err != nil {
			return nil, err
		}
		id = uuid.New()
		_, err = tx.Exec(ctx, `INSERT INTO comments (id,company_id,author_id,client_id,text,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,NOW(),NOW())`, id, companyID, author, clientID, text)
		if err != nil {
			return nil, err
		}
	} else {
		result, err := tx.Exec(ctx, `UPDATE comments SET text=$4,updated_at=NOW() WHERE id=$1 AND company_id=$2 AND client_id=$3`, id, companyID, clientID, text)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, ErrNotFound
		}
	}
	err = tx.QueryRow(ctx, `SELECT cm.id,cm.author_id,cm.text,concat_ws(' ',u.first_name,u.last_name),cm.created_at,cm.updated_at FROM comments cm JOIN users u ON u.id=cm.author_id WHERE cm.id=$1`, id).Scan(&c.ID, &c.AuthorID, &c.Text, &c.AuthorName, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) DeleteComment(ctx context.Context, companyID, clientID, id uuid.UUID) error {
	tx, _, err := r.clientTransaction(ctx, companyID, clientID)
	if err != nil {
		return err
	}
	defer rollback(tx)
	result, err := tx.Exec(ctx, `DELETE FROM comments WHERE id=$1 AND company_id=$2 AND client_id=$3`, id, companyID, clientID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

func (r *Repository) GetTags(ctx context.Context, companyID uuid.UUID) ([]Tag, error) {
	rows, err := r.db.Query(ctx, `SELECT id,name FROM tags WHERE company_id=$1 ORDER BY name,id`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := make([]Tag, 0)
	for rows.Next() {
		var t Tag
		if err = rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (r *Repository) CreateTag(ctx context.Context, companyID uuid.UUID, name string) (*Tag, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer rollback(tx)
	var company uuid.UUID
	if err = tx.QueryRow(ctx, `SELECT id FROM companies WHERE id=$1 FOR UPDATE`, companyID).Scan(&company); err != nil {
		return nil, err
	}
	var t Tag
	err = tx.QueryRow(ctx, `SELECT id,name FROM tags WHERE company_id=$1 AND lower(name)=lower($2) ORDER BY created_at,id LIMIT 1`, companyID, name).Scan(&t.ID, &t.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		t = Tag{ID: uuid.New(), Name: name}
		_, err = tx.Exec(ctx, `INSERT INTO tags (id,company_id,name,created_at) VALUES ($1,$2,$3,NOW())`, t.ID, companyID, name)
	}
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) SetTags(ctx context.Context, companyID, id uuid.UUID, tags []uuid.UUID) error {
	tx, _, err := r.clientTransaction(ctx, companyID, id)
	if err != nil {
		return err
	}
	defer rollback(tx)
	for _, tag := range tags {
		var exists bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tags WHERE id=$1 AND company_id=$2)`, tag, companyID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrInvalid
		}
	}
	if _, err = tx.Exec(ctx, `DELETE FROM client_tags WHERE client_id=$1`, id); err != nil {
		return err
	}
	for _, tag := range tags {
		if _, err = tx.Exec(ctx, `INSERT INTO client_tags (client_id,tag_id) VALUES ($1,$2)`, id, tag); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) GetClientTags(ctx context.Context, companyID, id uuid.UUID) ([]Tag, error) {
	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM clients WHERE company_id=$1 AND id=$2)`, companyID, id).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}
	rows, err := r.db.Query(ctx, `SELECT DISTINCT t.id,t.name FROM tags t JOIN client_tags ct ON ct.tag_id=t.id WHERE ct.client_id=$1 AND t.company_id=$2 ORDER BY t.name,t.id`, id, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := make([]Tag, 0)
	for rows.Next() {
		var t Tag
		if err = rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}
