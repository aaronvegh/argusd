package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_loadTokens(t *testing.T) {
	tokens, err := loadTokens("testdata/argusd.conf")
	if err != nil {
		t.Fatalf("got error: %v", err)
	}

	if len(tokens) != 2 {
		t.Fatalf("got %v want 2", len(tokens))
	}

	if tokens[0] != "716c468a-11eb-42cd-8140-27ef7c3986aa" {
		t.Fatalf("got error: %v", err)
	}

	if tokens[1] != "2814ca32-f211-4211-82d3-401e7cfb622b" {
		t.Fatalf("got error: %v", err)
	}
}

func newRequest(method, url string) *http.Request {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	return req
}

func Test_AuthenticationMiddleware_MissingHeader(t *testing.T) {
	r := newRequest("GET", "http://127.0.0.1/")
	rr := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	AuthenticationMiddleware("testdata/argusd.conf")(testHandler).ServeHTTP(rr, r)

	if got, want := rr.Code, http.StatusForbidden; got != want {
		t.Fatalf("bad status: got %v want %v", got, want)
	}
}

func Test_AuthenticationMiddleware_ValidToken1(t *testing.T) {
	r := newRequest("GET", "http://127.0.0.1/")
	r.Header.Set(tokenHeaderName, "716c468a-11eb-42cd-8140-27ef7c3986aa")
	rr := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	AuthenticationMiddleware("testdata/argusd.conf")(testHandler).ServeHTTP(rr, r)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("bad status: got %v want %v", got, want)
	}
}

func Test_AuthenticationMiddleware_ValidToken2(t *testing.T) {
	r := newRequest("GET", "http://127.0.0.1/")
	r.Header.Set(tokenHeaderName, "2814ca32-f211-4211-82d3-401e7cfb622b")
	rr := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	AuthenticationMiddleware("testdata/argusd.conf")(testHandler).ServeHTTP(rr, r)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("bad status: got %v want %v", got, want)
	}
}

func Test_AuthenticationMiddleware_InvalidToken(t *testing.T) {
	r := newRequest("GET", "http://127.0.0.1/")
	r.Header.Set(tokenHeaderName, "b29e00fd-3206-4c28-8169-4a765a8b4106")
	rr := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	AuthenticationMiddleware("testdata/argusd.conf")(testHandler).ServeHTTP(rr, r)

	if got, want := rr.Code, http.StatusForbidden; got != want {
		t.Fatalf("bad status: got %v want %v", got, want)
	}
}

func Test_AuthenticationMiddleware_MalformedToken(t *testing.T) {
	r := newRequest("GET", "http://127.0.0.1/")
	r.Header.Set(tokenHeaderName, "baconcheese")
	rr := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	AuthenticationMiddleware("/tmp/argusd.conf")(testHandler).ServeHTTP(rr, r)

	if got, want := rr.Code, http.StatusForbidden; got != want {
		t.Fatalf("bad status: got %v want %v", got, want)
	}
}
