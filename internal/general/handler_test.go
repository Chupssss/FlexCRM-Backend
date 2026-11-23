package general_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"FlexCRM-Backend/internal/general"
	"FlexCRM-Backend/internal/routers"
	"FlexCRM-Backend/internal/token"
	"github.com/google/uuid"
)

type stubRepository struct {
	calls             int
	limit             int
	userID, companyID uuid.UUID
	result            *general.Response
	err               error
}

func (s *stubRepository) Get(_ context.Context, userID, companyID uuid.UUID, limit int) (*general.Response, error) {
	s.calls++
	s.limit = limit
	s.userID = userID
	s.companyID = companyID
	return s.result, s.err
}

func TestGeneralHTTPContract(t *testing.T) {
	manager := token.NewTokenManager("test-only-secret")
	userID, companyID := uuid.New(), uuid.New()
	valid, err := manager.CreateToken(userID, companyID, "ADMIN", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	expired, err := manager.CreateToken(userID, companyID, "ADMIN", time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, path, cookie string
		repoErr            error
		want, limit        int
	}{
		{"default", "/api/general/", valid, nil, 200, 5},
		{"without trailing slash", "/api/general", valid, nil, 200, 5},
		{"minimum", "/api/general/?limit=1", valid, nil, 200, 1},
		{"maximum", "/api/general/?limit=50", valid, nil, 200, 50},
		{"zero", "/api/general/?limit=0", valid, nil, 400, 0},
		{"negative", "/api/general/?limit=-1", valid, nil, 400, 0},
		{"too large", "/api/general/?limit=51", valid, nil, 400, 0},
		{"not integer", "/api/general/?limit=abc", valid, nil, 400, 0},
		{"empty", "/api/general/?limit=", valid, nil, 400, 0},
		{"duplicate", "/api/general/?limit=1&limit=2", valid, nil, 400, 0},
		{"missing token", "/api/general/", "", nil, 401, 0},
		{"invalid token", "/api/general/", "invalid", nil, 401, 0},
		{"expired token", "/api/general/", expired, nil, 401, 0},
		{"inactive user", "/api/general/", valid, general.ErrForbidden, 403, 5},
		{"database error", "/api/general/", valid, errors.New("private SQL failure"), 500, 5},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &stubRepository{result: &general.Response{FirstName: "Ivan", Role: "ADMIN", Tasks: []general.Task{}, Activities: []general.Activity{}}, err: tt.repoErr}
			mux := http.NewServeMux()
			routers.SetupRouter(mux, routers.Dependencies{GeneralHandler: general.NewHandler(general.NewService(repo)), Tokenmanager: manager})
			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "access-token", Value: tt.cookie})
			}
			res := httptest.NewRecorder()
			mux.ServeHTTP(res, req)
			if res.Code != tt.want {
				t.Fatalf("status %d want %d: %s", res.Code, tt.want, res.Body.String())
			}
			if res.Header().Get("Content-Type") != "application/json" {
				t.Fatal("expected JSON")
			}
			if tt.limit == 0 {
				if repo.calls != 0 {
					t.Fatal("repository called for rejected request")
				}
				return
			}
			if repo.calls != 1 || repo.limit != tt.limit || repo.userID != userID || repo.companyID != companyID {
				t.Fatalf("wrong repository arguments: %+v", repo)
			}
			var data map[string]json.RawMessage
			if err := json.Unmarshal(res.Body.Bytes(), &data); err != nil {
				t.Fatal(err)
			}
			if tt.want == 200 {
				for _, field := range []string{"first_name", "last_name", "role", "count_of_all_deals", "count_clients", "count_tasks", "tasks", "activities"} {
					if _, ok := data[field]; !ok {
						t.Fatalf("missing Swagger field %s", field)
					}
				}
				if string(data["tasks"]) != "[]" || string(data["activities"]) != "[]" {
					t.Fatal("empty lists must be arrays")
				}
			} else {
				if _, ok := data["code"]; !ok {
					t.Fatal("missing error code")
				}
				if _, ok := data["message"]; !ok {
					t.Fatal("missing error message")
				}
			}
		})
	}
}
