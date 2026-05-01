package auth

import "net/http"

// extractSessionToken parses the "session" cookie value from a raw Cookie header.
func extractSessionToken(rawCookie string) string {
	header := http.Header{}
	header.Add("Cookie", rawCookie)
	req := http.Request{Header: header}
	c, err := req.Cookie("session")
	if err != nil {
		return ""
	}
	return c.Value
}
