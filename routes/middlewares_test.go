package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"server-go/common"
)

func TestAdminMiddlewareRejectsEmptyConfiguredAdminToken(t *testing.T) {
	origConfig := common.Config
	common.Config = &common.ConfigStr{AdminToken: ""}
	t.Cleanup(func() {
		common.Config = origConfig
	})

	called := false
	handler := AdminMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/reviewdb/admin/reports", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if called {
		t.Fatal("admin handler was called with empty request token and empty configured admin token")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAdminMiddlewareAllowsNonEmptyConfiguredAdminToken(t *testing.T) {
	origConfig := common.Config
	common.Config = &common.ConfigStr{AdminToken: "secret-admin-token"}
	t.Cleanup(func() {
		common.Config = origConfig
	})

	called := false
	handler := AdminMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/reviewdb/admin/reports", nil)
	req.Header.Set("Authorization", "secret-admin-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("admin handler was not called with matching non-empty configured admin token")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}
