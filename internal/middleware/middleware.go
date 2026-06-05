// Package middleware holds cross-cutting HTTP middleware: auth, logging, CORS.
package middleware

import (
	"brewme/internal/database"
	"brewme/internal/utils"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/redis/go-redis/v9"
	"golang.org/x/net/html"
)

// StripHTMLTags removes any HTML tags from the input string
// and returns only the text content
func StripHTMLTags(input string) string {
	doc, err := html.Parse(bytes.NewBufferString(input))
	if err != nil {
		return html.EscapeString(input) // Fallback to escaping if parsing fails
	}

	var output bytes.Buffer
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			output.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return strings.TrimSpace(output.String())
}

// SanitizeHTML removes potentially dangerous HTML tags and attributes
// It first tries to strip tags, falling back to standard HTML escaping
func SanitizeHTML(input string) string {
	stripped := StripHTMLTags(input)

	if stripped == "" {
		return html.EscapeString(input) // Fallback to escaping
	}
	return stripped
}

func SessionMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// 1. Extract token
		tokenStr, _ := utils.GetTokenFromHeader(r) // reuse your existing helper
		if tokenStr == "" {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}

		// 2. Validate JWT signature & expiry
		claims, err := utils.ValidateToken(tokenStr)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// 3. Check Redis session exists
		sessionKey := fmt.Sprintf("session:%s", claims.Email)
		sessionJSON, err := database.Redis.Get(r.Context(), sessionKey).Result()
		if err == redis.Nil {
			http.Error(w, "Session expired or logged out", http.StatusUnauthorized)
			return
		} else if err != nil {
			http.Error(w, "Session store error", http.StatusInternalServerError)
			return
		}

		// 4. Verify stored token matches (blocks old tokens after re-login)
		var sessionData map[string]interface{}
		if err := json.Unmarshal([]byte(sessionJSON), &sessionData); err != nil {
			http.Error(w, "Session corrupted", http.StatusInternalServerError)
			return
		}
		if sessionData["token"] != tokenStr {
			http.Error(w, "Token mismatch — please login again", http.StatusUnauthorized)
			return
		}

		// 5. Pass email downstream via context
		ctx := context.WithValue(r.Context(), "email", claims.Email)
		next(w, r.WithContext(ctx))
	}
}
