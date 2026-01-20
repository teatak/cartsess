package cartsess

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStandardMiddleware(t *testing.T) {
	store := NewMemoryStore()
	cookieName := "test-session"
	middleware := NewManager(cookieName, store)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manager := Default(r.Context())
		if manager == nil {
			t.Fatal("session manager not found in context")
		}

		// Check name
		if manager.Name() != cookieName {
			t.Errorf("expected cookie name %s, got %s", cookieName, manager.Name())
		}

		// Set value
		count := 0
		if val, _ := manager.Get("count"); val != nil {
			count = val.(int)
		}
		count++
		manager.Set("count", count)

		w.Write([]byte(fmt.Sprintf("count:%d", count)))
	})

	wrappedHandler := middleware(handler)

	// First request: should set cookie and count=1
	req1 := httptest.NewRequest("GET", "/", nil)
	rec1 := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec1, req1)

	if rec1.Body.String() != "count:1" {
		t.Errorf("expected body 'count:1', got '%s'", rec1.Body.String())
	}

	cookies := rec1.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie")
	}
	sessionCookie := cookies[0]
	if sessionCookie.Name != cookieName {
		t.Errorf("expected cookie name %s, got %s", cookieName, sessionCookie.Name)
	}

	// Second request: should have count=2
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.AddCookie(sessionCookie)
	rec2 := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec2, req2)

	if rec2.Body.String() != "count:2" {
		t.Errorf("expected body 'count:2', got '%s'", rec2.Body.String())
	}
}

func TestGetByName(t *testing.T) {
	store1 := NewMemoryStore()
	store2 := NewMemoryStore()

	mw1 := NewManager("sess1", store1)
	mw2 := NewManager("sess2", store2)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s1 := GetByName(r.Context(), "sess1")
		s2 := GetByName(r.Context(), "sess2")

		s1.Set("k1", "v1")
		s2.Set("k2", "v2")
	})

	// Chain middlewares
	chain := mw1(mw2(handler))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	chain.ServeHTTP(rec, req)

	// Verify logic ran without panic
	if rec.Result().StatusCode != 200 {
		t.Errorf("expected 200 OK")
	}
}
