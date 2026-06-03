// Package middleware holds cross-cutting HTTP middleware: auth, logging, CORS.
package middleware

import (
	"bytes"
	"strings"

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
