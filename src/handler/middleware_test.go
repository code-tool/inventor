package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func newTestMiddleware(t *testing.T) *SDTargetsMiddleware {
	t.Helper()
	con := newTestClient(t)
	return &SDTargetsMiddleware{
		SDTargets: &SDTargets{Items: make(map[uuid.UUID]StaticConfig)},
		Context:   context.Background(),
		Client:    con,
		ApiToken:  "secret",
		SdToken:   "",
		TTL:       60,
	}
}

func doRequest(handlerFn http.HandlerFunc, method, target string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	handlerFn(rec, req)
	return rec
}

func TestHandleSDTarget_ForbiddenWithoutToken(t *testing.T) {
	s := newTestMiddleware(t)
	rec := doRequest(s.HandleSDTarget, http.MethodGet, "/target", nil, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without x-api-token, got %d", rec.Code)
	}
}

func TestHandleSDTarget_ForbiddenWithWrongToken(t *testing.T) {
	s := newTestMiddleware(t)
	rec := doRequest(s.HandleSDTarget, http.MethodGet, "/target", nil, map[string]string{"x-api-token": "wrong"})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 with wrong x-api-token, got %d", rec.Code)
	}
}

func TestHandleSDTarget_MethodNotAllowed(t *testing.T) {
	s := newTestMiddleware(t)
	rec := doRequest(s.HandleSDTarget, http.MethodPost, "/target", nil, map[string]string{"x-api-token": "secret"})
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for unsupported method, got %d", rec.Code)
	}
}

func TestHandleInsertAndGetByID(t *testing.T) {
	s := newTestMiddleware(t)

	insertBody, _ := json.Marshal(StaticConfigDocument{
		SDTarget: StaticConfig{Targets: []string{"10.0.0.1:9100"}, Labels: map[string]string{"job": "node"}},
	})
	putRec := doRequest(s.HandleSDTarget, http.MethodPut, "/target", insertBody, map[string]string{"x-api-token": "secret"})
	if putRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on insert, got %d body=%s", putRec.Code, putRec.Body.String())
	}
	var idDoc IDDocument
	if err := json.Unmarshal(putRec.Body.Bytes(), &idDoc); err != nil {
		t.Fatalf("failed to decode insert response: %v", err)
	}

	getBody, _ := json.Marshal(IDDocument{ID: idDoc.ID})
	getRec := doRequest(s.HandleSDTarget, http.MethodGet, "/target", getBody, map[string]string{"x-api-token": "secret"})
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on get, got %d body=%s", getRec.Code, getRec.Body.String())
	}
	var doc StaticConfigDocument
	if err := json.Unmarshal(getRec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("failed to decode get response: %v", err)
	}
	if len(doc.SDTarget.Targets) != 1 || doc.SDTarget.Targets[0] != "10.0.0.1:9100" {
		t.Errorf("unexpected targets in response: %+v", doc.SDTarget.Targets)
	}
}

func TestHandleGetByID_NotFound(t *testing.T) {
	s := newTestMiddleware(t)
	body, _ := json.Marshal(IDDocument{ID: uuid.New()})
	rec := doRequest(s.HandleSDTarget, http.MethodGet, "/target", body, map[string]string{"x-api-token": "secret"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing id, got %d", rec.Code)
	}
}

func TestHandleDelete_NotFound(t *testing.T) {
	s := newTestMiddleware(t)
	body, _ := json.Marshal(IDDocument{ID: uuid.New()})
	rec := doRequest(s.HandleSDTarget, http.MethodDelete, "/target", body, map[string]string{"x-api-token": "secret"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 deleting a missing id, got %d", rec.Code)
	}
}

func TestHandleDelete_Success(t *testing.T) {
	s := newTestMiddleware(t)
	id, err := s.SDTargets.Insert(StaticConfig{Targets: []string{"10.0.0.1:9100"}}, s.Context, s.Client, s.TTL)
	if err != nil {
		t.Fatalf("unexpected error seeding target: %v", err)
	}

	body, _ := json.Marshal(IDDocument{ID: id})
	rec := doRequest(s.HandleSDTarget, http.MethodDelete, "/target", body, map[string]string{"x-api-token": "secret"})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleDiscover_MethodNotAllowed(t *testing.T) {
	s := newTestMiddleware(t)
	rec := doRequest(s.HandleDiscover, http.MethodPost, "/discover", nil, nil)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for unsupported method, got %d", rec.Code)
	}
}

func TestHandleDiscover_TokenNotEnforcedWhenEmpty(t *testing.T) {
	s := newTestMiddleware(t)
	rec := doRequest(s.HandleDiscover, http.MethodGet, "/discover", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 when SdToken is empty, got %d", rec.Code)
	}
}

func TestHandleDiscover_TokenEnforcedWhenSet(t *testing.T) {
	s := newTestMiddleware(t)
	s.SdToken = "sdsecret"

	rec := doRequest(s.HandleDiscover, http.MethodGet, "/discover", nil, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without x-sd-token, got %d", rec.Code)
	}

	rec = doRequest(s.HandleDiscover, http.MethodGet, "/discover", nil, map[string]string{"x-sd-token": "sdsecret"})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with correct x-sd-token, got %d", rec.Code)
	}
}

func TestHandleGetAll_ExpandsModules(t *testing.T) {
	s := newTestMiddleware(t)
	_, err := s.SDTargets.Insert(StaticConfig{
		Targets: []string{"10.0.0.1:9100"},
		Labels:  map[string]string{"job": "node"},
		Modules: []string{"mod1", "mod2"},
	}, s.Context, s.Client, s.TTL)
	if err != nil {
		t.Fatalf("unexpected error seeding target: %v", err)
	}

	rec := doRequest(s.HandleDiscover, http.MethodGet, "/discover", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var res []HttpSD
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected one entry per module (2), got %d", len(res))
	}
	seen := map[string]bool{}
	for _, item := range res {
		seen[item.Labels["__meta_inventor_sd_module"]] = true
		if item.Labels["job"] != "node" {
			t.Errorf("expected original labels to be preserved, got %+v", item.Labels)
		}
	}
	if !seen["mod1"] || !seen["mod2"] {
		t.Errorf("expected both modules to be represented, got %+v", seen)
	}
}

func TestHandleGetByGroupName_MissingParam(t *testing.T) {
	s := newTestMiddleware(t)
	rec := doRequest(s.HandleDiscoverGroup, http.MethodGet, "/group", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when name param is missing, got %d", rec.Code)
	}
}

func TestHandleGetByGroupName_FiltersByGroup(t *testing.T) {
	s := newTestMiddleware(t)
	if _, err := s.SDTargets.Insert(StaticConfig{Targets: []string{"10.0.0.1:9100"}, Group: "a"}, s.Context, s.Client, s.TTL); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := s.SDTargets.Insert(StaticConfig{Targets: []string{"10.0.0.2:9100"}, Group: "b"}, s.Context, s.Client, s.TTL); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec := doRequest(s.HandleDiscoverGroup, http.MethodGet, "/group?name=a", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var res []HttpSD
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(res) != 1 || res[0].Targets[0] != "10.0.0.1:9100" {
		t.Fatalf("expected only group `a` target, got %+v", res)
	}
}

func TestHealthCheck(t *testing.T) {
	s := newTestMiddleware(t)
	rec := doRequest(s.HealthCheck, http.MethodGet, "/healthcheck", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "OK" {
		t.Errorf("expected body %q, got %q", "OK", rec.Body.String())
	}
}
