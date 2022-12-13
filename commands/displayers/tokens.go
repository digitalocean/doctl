package displayers

import (
	"io"
	"time"

	"github.com/digitalocean/doctl/do"
)

type Tokens struct {
	Tokens          []do.Token
	WithAccessToken bool
}

var _ Displayable = &Tokens{}

func (t *Tokens) JSON(out io.Writer) error {
	return writeJSON(t.Tokens, out)
}

func (t *Tokens) Cols() []string {
	cols := []string{
		"ID", "Name", "Scopes", "ExpiresAt",
	}

	// We only return the the access token in the recreate response.
	if t.WithAccessToken {
		cols = append(cols, "AccessToken")
	} else {
		cols = append(cols, "LastUsedAt", "CreatedAt")
	}

	return cols
}

func (t *Tokens) ColMap() map[string]string {
	return map[string]string{
		"ID":          "ID",
		"Name":        "Name",
		"Scopes":      "Scopes",
		"ExpiresAt":   "Expires At",
		"CreatedAt":   "Created At",
		"LastUsedAt":  "Last Used At",
		"AccessToken": "Access Token",
	}
}

func (t *Tokens) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(t.Tokens))

	for _, t := range t.Tokens {
		m := map[string]interface{}{
			"ID":          t.ID,
			"Name":        t.Name,
			"Scopes":      t.Scopes,
			"ExpiresAt":   "Never",
			"CreatedAt":   t.CreatedAt.String(),
			"LastUsedAt":  t.LastUsedAt,
			"AccessToken": t.AccessToken,
		}

		if t.ExpirySeconds != nil {
			m["ExpiresAt"] = t.CreatedAt.Add(time.Duration(*t.ExpirySeconds) * time.Second)
		}
		out = append(out, m)
	}

	return out
}

type TokenScopes struct {
	TokenScopes []do.TokenScope
}

var _ Displayable = &TokenScopes{}

func (s *TokenScopes) JSON(out io.Writer) error {
	return writeJSON(s.TokenScopes, out)
}

func (s *TokenScopes) Cols() []string {
	return []string{"Name"}
}

func (s *TokenScopes) ColMap() map[string]string {
	return map[string]string{
		"Name": "Name",
	}
}

func (s *TokenScopes) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0)

	for _, s := range s.TokenScopes {
		m := map[string]interface{}{
			"Name": s.Name,
		}

		out = append(out, m)
	}

	return out
}
