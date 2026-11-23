package clients_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"FlexCRM-Backend/internal/clients"
	"FlexCRM-Backend/internal/routers"
	"FlexCRM-Backend/internal/token"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func profileRouter(manager *token.TokenManager, handler *clients.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	routers.SetupRouter(mux, routers.Dependencies{ClientsHandler: handler, Tokenmanager: manager})
	return mux
}

func TestProfileRejectsInvalidRequests(t *testing.T) {
	manager := token.NewTokenManager("test-only-secret")
	valid, err := manager.CreateToken(uuid.New(), uuid.New(), "ADMIN", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	expired, err := manager.CreateToken(uuid.New(), uuid.New(), "ADMIN", time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	mux := profileRouter(manager, clients.NewHandler(nil))
	for _, prefix := range []string{"/api/clients/", "/api/v1/clients/"} {
		for _, suffix := range []string{"", "/"} {
			for _, tt := range []struct {
				name, id, cookie string
				want             int
				code             string
			}{
				{"missing cookie", uuid.NewString(), "", 401, "UNAUTHORIZED"},
				{"invalid cookie", uuid.NewString(), "invalid", 401, "UNAUTHORIZED"},
				{"expired cookie", uuid.NewString(), expired, 401, "UNAUTHORIZED"},
				{"invalid id", "bad-id", valid, 400, "INVALID_CLIENT_ID"},
				{"nil id", uuid.Nil.String(), valid, 400, "INVALID_CLIENT_ID"},
			} {
				t.Run(prefix+tt.name+suffix, func(t *testing.T) {
					req := httptest.NewRequest("GET", prefix+tt.id+suffix, nil)
					if tt.cookie != "" {
						req.AddCookie(&http.Cookie{Name: "access-token", Value: tt.cookie})
					}
					res := httptest.NewRecorder()
					mux.ServeHTTP(res, req)
					if res.Code != tt.want {
						t.Fatalf("got %d want %d: %s", res.Code, tt.want, res.Body.String())
					}
					var data map[string]string
					if err := json.Unmarshal(res.Body.Bytes(), &data); err != nil {
						t.Fatal(err)
					}
					if data["code"] != tt.code {
						t.Fatalf("wrong error: %v", data)
					}
				})
			}
		}
	}
}

func TestProfilePostgres(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL for PostgreSQL integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "profile_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	migrations, err := filepath.Glob("../../migrations/*.up.sql")
	if err != nil || len(migrations) == 0 {
		t.Fatalf("migrations: %v", err)
	}
	for _, name := range migrations {
		sql, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		exec(string(sql))
	}
	companyA, companyB, userA, userB := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	for _, pair := range [][2]uuid.UUID{{companyA, userA}, {companyB, userB}} {
		exec(`INSERT INTO companies(id,name,created_at,updated_at) VALUES($1,'Test',NOW(),NOW())`, pair[0])
		exec(`INSERT INTO users(id,company_id,email,password_hash,first_name,last_name,role,created_at,updated_at) VALUES($1,$2,$3,'not-used','Manager','One','ADMIN',NOW(),NOW())`, pair[1], pair[0], pair[1].String()+"@example.com")
	}
	organization, person, emptyCompany := uuid.New(), uuid.New(), uuid.New()
	exec(`INSERT INTO clients(id,company_id,created_by,responsible_user_id,client_type,first_name,last_name,company_name,email,phone,description,created_at,updated_at) VALUES($1,$2,$3,$3,'COMPANY','Old','Name','Client Organization','info@example.com','123','Description',NOW(),NOW())`, organization, companyA, userA)
	exec(`INSERT INTO clients(id,company_id,created_by,client_type,first_name,last_name,created_at,updated_at) VALUES($1,$2,$3,'PERSON','Person','Surname',NOW(),NOW())`, person, companyA, userA)
	exec(`INSERT INTO clients(id,company_id,created_by,client_type,first_name,last_name,company_name,created_at,updated_at) VALUES($1,$2,$3,'COMPANY','Representative','Surname','Empty organization',NOW(),NOW())`, emptyCompany, companyA, userA)
	now := time.Now().UTC().Truncate(time.Second)
	primary, secondary1, secondary2 := uuid.New(), uuid.New(), uuid.New()
	exec(`INSERT INTO contacts(id,client_id,first_name,last_name,is_primary,created_at,updated_at) VALUES($1,$2,'Primary','Representative',true,$3,$3)`, primary, organization, now)
	exec(`INSERT INTO contacts(id,client_id,first_name,last_name,position,email,phone,created_at,updated_at) VALUES($1,$2,'Secondary','One','Accountant','secondary@example.com','456',$3,$3)`, secondary1, organization, now.Add(time.Minute))
	exec(`INSERT INTO contacts(id,client_id,first_name,created_at,updated_at) VALUES($1,$2,'Secondary two',$3,$3)`, secondary2, organization, now.Add(2*time.Minute))
	// Even legacy inconsistent contact rows must not appear on a PERSON profile.
	exec(`INSERT INTO contacts(client_id,first_name,created_at,updated_at) VALUES($1,'Legacy contact',NOW(),NOW())`, person)
	tagA, tagB, foreignTag := uuid.New(), uuid.New(), uuid.New()
	exec(`INSERT INTO tags(id,company_id,name,created_at) VALUES($1,$2,'B2B',NOW()),($3,$2,'VIP',NOW()),($4,$5,'Foreign',NOW())`, tagA, companyA, tagB, foreignTag, companyB)
	exec(`INSERT INTO client_tags(client_id,tag_id) VALUES($1,$2),($1,$2),($1,$3),($1,$4)`, organization, tagA, tagB, foreignTag)
	exec(`INSERT INTO comments(company_id,author_id,client_id,text,created_at,updated_at) VALUES($1,$2,$3,'Old comment',$4,$4)`, companyA, userA, organization, now)
	exec(`INSERT INTO comments(company_id,author_id,client_id,text,created_at,updated_at) VALUES($1,$2,$3,'Latest comment',$4,$4)`, companyA, userA, organization, now.Add(time.Minute))
	exec(`INSERT INTO comments(company_id,author_id,client_id,text,created_at,updated_at) VALUES($1,$2,$3,'Foreign comment',$4,$4)`, companyB, userB, organization, now.Add(time.Hour))
	manager := token.NewTokenManager("test-only-secret")
	mux := profileRouter(manager, clients.NewHandler(clients.NewService(clients.NewRepository(pool))))
	request := func(path string, user, company uuid.UUID, want int) *httptest.ResponseRecorder {
		t.Helper()
		cookie, err := manager.CreateToken(user, company, "ADMIN", time.Now().Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("GET", path, nil)
		req.AddCookie(&http.Cookie{Name: "access-token", Value: cookie})
		res := httptest.NewRecorder()
		mux.ServeHTTP(res, req)
		if res.Code != want {
			t.Fatalf("%s: got %d want %d: %s", path, res.Code, want, res.Body.String())
		}
		return res
	}
	decode := func(res *httptest.ResponseRecorder) clients.Profile {
		t.Helper()
		var profile clients.Profile
		if err := json.Unmarshal(res.Body.Bytes(), &profile); err != nil {
			t.Fatal(err)
		}
		return profile
	}
	var profile clients.Profile
	for _, prefix := range []string{"/api/clients/", "/api/v1/clients/"} {
		for _, suffix := range []string{"", "/"} {
			res := request(prefix+organization.String()+suffix, userA, companyA, 200)
			profile = decode(res)
			if profile.ID != organization || profile.Type != "COMPANY" || profile.FirstName == nil || *profile.FirstName != "Primary" || profile.LastName == nil || *profile.LastName != "Representative" {
				t.Fatalf("wrong representative: %+v", profile)
			}
			if len(profile.Contacts) != 2 || profile.Contacts[0].ID != secondary1 || profile.Contacts[1].ID != secondary2 {
				t.Fatalf("wrong contacts: %+v", profile.Contacts)
			}
			if profile.ResponsibleName == nil || *profile.ResponsibleName != "Manager One" || profile.Description == nil || *profile.Description != "Description" {
				t.Fatalf("missing profile fields: %+v", profile)
			}
			if profile.LastComment == nil || profile.LastComment.Text != "Latest comment" || profile.LastComment.AuthorName != "Manager One" {
				t.Fatalf("wrong comment: %+v", profile.LastComment)
			}
			if strings.Join(profile.Tags, ",") != "B2B,VIP" {
				t.Fatalf("wrong tags: %v", profile.Tags)
			}
			// ClientDetail and ClientContact prohibit extra fields in the current Swagger.
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(res.Body.Bytes(), &fields); err != nil {
				t.Fatal(err)
			}
			expected := []string{"id", "client_type", "first_name", "last_name", "company_name", "phone", "email", "responsible_user_name", "description", "last_comment", "tags", "contacts"}
			if len(fields) != len(expected) {
				t.Fatalf("extra/missing profile fields: %v", fields)
			}
			for _, name := range expected {
				if _, ok := fields[name]; !ok {
					t.Fatalf("missing %s", name)
				}
			}
			var contacts []map[string]json.RawMessage
			if err := json.Unmarshal(fields["contacts"], &contacts); err != nil {
				t.Fatal(err)
			}
			for _, contact := range contacts {
				if len(contact) != 6 {
					t.Fatalf("unexpected contact fields: %v", contact)
				}
			}
			request(prefix+organization.String()+suffix, userB, companyB, 404)
		}
	}
	personProfile := decode(request("/api/clients/"+person.String()+"/", userA, companyA, 200))
	if personProfile.Type != "PERSON" || personProfile.CompanyName != nil || personProfile.Contacts == nil || len(personProfile.Contacts) != 0 || personProfile.Tags == nil || personProfile.LastComment != nil || personProfile.FirstName == nil || *personProfile.FirstName != "Person" {
		t.Fatalf("wrong person profile: %+v", personProfile)
	}
	empty := decode(request("/api/clients/"+emptyCompany.String()+"/", userA, companyA, 200))
	if empty.Contacts == nil || len(empty.Contacts) != 0 || empty.Tags == nil || len(empty.Tags) != 0 || empty.LastComment != nil {
		t.Fatalf("wrong empty company profile: %+v", empty)
	}
	missing := request("/api/clients/"+uuid.NewString(), userA, companyA, 404)
	if !strings.Contains(missing.Body.String(), "CLIENT_NOT_FOUND") {
		t.Fatal("wrong missing client error")
	}
	pool.Close()
	failure := request("/api/clients/"+organization.String(), userA, companyA, 500)
	if !strings.Contains(failure.Body.String(), "INTERNAL_ERROR") || strings.Contains(failure.Body.String(), "closed pool") {
		t.Fatal("wrong database error response")
	}
}
