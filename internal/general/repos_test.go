package general_test

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

	"FlexCRM-Backend/internal/general"
	"FlexCRM-Backend/internal/routers"
	"FlexCRM-Backend/internal/token"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestGeneralPostgres(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "general_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
		exec(`INSERT INTO users(id,company_id,email,password_hash,first_name,last_name,role,created_at,updated_at) VALUES($1,$2,$3,'not-used','Ivan',NULL,'ADMIN',NOW(),NOW())`, pair[1], pair[0], pair[1].String()+"@example.com")
	}
	manager := token.NewTokenManager("test-only-secret")
	mux := http.NewServeMux()
	routers.SetupRouter(mux, routers.Dependencies{GeneralHandler: general.NewHandler(general.NewService(general.NewRepository(pool))), Tokenmanager: manager})
	get := func(user, company uuid.UUID, path string, want int) general.Response {
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
			t.Fatalf("status %d want %d: %s", res.Code, want, res.Body.String())
		}
		var data general.Response
		if want == 200 {
			if err := json.Unmarshal(res.Body.Bytes(), &data); err != nil {
				t.Fatal(err)
			}
		}
		return data
	}
	empty := get(userA, companyA, "/api/general/", 200)
	if empty.FirstName != "Ivan" || empty.LastName != "" || empty.Role != "ADMIN" || empty.CountTasks != 0 || empty.CountClients != 0 || empty.CountOfAllDeals != 0 || empty.Tasks == nil || empty.Activities == nil {
		t.Fatalf("empty response: %+v", empty)
	}
	client, pipeline, stage := uuid.New(), uuid.New(), uuid.New()
	exec(`INSERT INTO clients(id,company_id,created_by,client_type,company_name,created_at,updated_at) VALUES($1,$2,$3,'COMPANY','Company A',NOW(),NOW())`, client, companyA, userA)
	exec(`INSERT INTO pipelines(id,company_id,name,created_at,updated_at) VALUES($1,$2,'Pipeline',NOW(),NOW())`, pipeline, companyA)
	exec(`INSERT INTO pipeline_stages(id,pipeline_id,name,position,stage_type,created_at,updated_at) VALUES($1,$2,'New',0,'OPEN',NOW(),NOW())`, stage, pipeline)
	exec(`INSERT INTO deals(company_id,client_id,pipeline_id,stage_id,created_by,title,created_at,updated_at) VALUES($1,$2,$3,$4,$5,'Deal',NOW(),NOW())`, companyA, client, pipeline, stage, userA)
	now := time.Now().UTC().Truncate(time.Second)
	// Insert out of order; expired deadlines remain actionable and must appear first.
	for _, task := range []struct {
		title, status  string
		due, completed any
	}{
		{"no deadline", "OPEN", nil, nil},
		{"later", "IN_PROGRESS", now.Add(time.Hour), nil},
		{"overdue", "OPEN", now.Add(-time.Hour), nil},
		{"closed by status", "DONE", now.Add(-time.Hour * 3), nil},
		{"closed by date", "OPEN", now.Add(-time.Hour * 4), now},
		{"cancelled", "CANCELLED", now.Add(-time.Hour * 5), nil},
	} {
		exec(`INSERT INTO tasks(company_id,created_by,title,status,priority,due_at,completed_at,created_at,updated_at) VALUES($1,$2,$3,$4,'MEDIUM',$5,$6,NOW(),NOW())`, companyA, userA, task.title, task.status, task.due, task.completed)
	}
	exec(`INSERT INTO tasks(company_id,created_by,title,status,priority,created_at,updated_at) VALUES($1,$2,'foreign task','OPEN','HIGH',NOW(),NOW())`, companyB, userB)
	for i := 0; i < 7; i++ {
		exec(`INSERT INTO activities(company_id,actor_id,entity_type,entity_id,action,created_at) VALUES($1,$2,'CLIENT',$3,'CLIENT_CREATED',$4)`, companyA, userA, client, now.Add(-time.Duration(i+1)*time.Minute))
	}
	exec(`INSERT INTO activities(company_id,actor_id,entity_type,entity_id,action,created_at) VALUES($1,NULL,'CLIENT',$2,'SYSTEM_EVENT',$3)`, companyA, client, now)
	exec(`INSERT INTO activities(company_id,actor_id,entity_type,entity_id,action,created_at) VALUES($1,$2,'CLIENT',$3,'FOREIGN_EVENT',$4)`, companyB, userB, uuid.New(), now.Add(time.Minute))
	data := get(userA, companyA, "/api/general/", 200)
	if data.CountClients != 1 || data.CountOfAllDeals != 1 || data.CountTasks != 3 || len(data.Activities) != 5 || len(data.Tasks) != 3 {
		t.Fatalf("wrong totals: %+v", data)
	}
	if data.Tasks[0].Title != "overdue" || data.Tasks[1].Title != "later" || data.Tasks[2].DueAt != nil {
		t.Fatalf("task ordering: %+v", data.Tasks)
	}
	if data.Activities[0].ActorName != "Система" || data.Activities[1].ActorName != "Ivan" {
		t.Fatalf("actor names: %+v", data.Activities)
	}
	limited := get(userA, companyA, "/api/general/?limit=1", 200)
	if len(limited.Tasks) != 1 || len(limited.Activities) != 1 || limited.CountTasks != 3 {
		t.Fatal("limit must cap arrays, not totals")
	}
	other := get(userB, companyB, "/api/general/", 200)
	if other.CountClients != 0 || other.CountOfAllDeals != 0 || other.CountTasks != 1 || len(other.Activities) != 1 || other.Activities[0].Action != "FOREIGN_EVENT" {
		t.Fatalf("company isolation: %+v", other)
	}
	get(userA, companyB, "/api/general/", 403)
	exec(`UPDATE users SET is_active=false WHERE id=$1`, userA)
	get(userA, companyA, "/api/general/", 403)
}
