package oauth2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestResponses(t *testing.T) {
	for _, body := range []string{`{"access_token":"","expires_in":3600}`, `{"access_token":"x","expires_in":0}`, `{`, strings.Repeat("x", 65537)} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
		s := NewTokenStore("a", "b", "c", srv.URL)
		if _, e := s.Token(); e == nil {
			t.Fatal(body[:1])
		}
		srv.Close()
	}
}
func TestStatusAndDeadline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bad" {
			w.WriteHeader(500)
			return
		}
		select {
		case <-r.Context().Done():
		case <-time.After(time.Second):
		}
		w.Write([]byte(`{"access_token":"x","expires_in":30}`))
	}))
	defer srv.Close()
	if _, e := NewTokenStore("a", "b", "c", srv.URL+"/bad").Token(); e == nil {
		t.Fatal("status")
	}
	ctx, c := context.WithTimeout(context.Background(), time.Millisecond)
	defer c()
	if _, e := NewTokenStore("a", "b", "c", srv.URL).TokenContext(ctx); e == nil {
		t.Fatal("deadline")
	}
}
func TestShortExpiry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"access_token":"x","expires_in":30}`)) }))
	defer srv.Close()
	s := NewTokenStore("a", "b", "c", srv.URL)
	tok, e := s.Token()
	if tok != "x" || e != nil || !s.expiresAt.After(time.Now()) {
		t.Fatal(tok, e)
	}
}
