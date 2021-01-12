package main

import (
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

const tokenHeaderName = "X-Argus-Token"

var tokenMatcher = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

func loadTokens(path string) ([]string, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, token := range strings.Split(string(contents), "\n") {
		if tokenMatcher.MatchString(token) {
			result = append(result, token)
		}
	}

	return result, nil
}

func AuthenticationMiddleware(path string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token := r.Header.Get(tokenHeaderName); tokenMatcher.MatchString(token) {
				tokens, err := loadTokens(path)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
					return
				}

				if _, found := FindInArray(tokens, token); found {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		})
	}
}
