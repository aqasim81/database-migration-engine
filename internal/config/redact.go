package config

import (
	"net/url"
	"strings"
)

// RedactURL replaces the password in a PostgreSQL connection URL with "***".
// If the URL cannot be parsed or has no password, it is returned unchanged.
func RedactURL(raw string) string {
	if raw == "" {
		return ""
	}

	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	if u.User == nil {
		return raw
	}

	if _, hasPassword := u.User.Password(); !hasPassword {
		return raw
	}

	// Find the userinfo section between "://" and "@" in the raw string,
	// then replace the password portion (after "username:") with "***".
	schemeEnd := strings.Index(raw, "://")
	if schemeEnd < 0 {
		return raw
	}

	afterScheme := schemeEnd + len("://")

	atIdx := strings.Index(raw[afterScheme:], "@")
	if atIdx < 0 {
		return raw
	}

	userinfo := raw[afterScheme : afterScheme+atIdx]
	colonIdx := strings.Index(userinfo, ":")

	if colonIdx < 0 {
		return raw
	}

	redacted := raw[:afterScheme] + userinfo[:colonIdx+1] + "***" + raw[afterScheme+atIdx:]

	return redacted
}
