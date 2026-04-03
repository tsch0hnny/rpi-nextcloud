package ui

import (
	"regexp"
	"strings"
)

var mysqlIdentRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
var phpSizeRe = regexp.MustCompile(`^[0-9]+[mMgGkK]?$`)
var domainRe = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*$`)
var pathRe = regexp.MustCompile(`^/[a-zA-Z0-9._/-]+$`)

// ValidateDBName checks if a string is a valid MySQL identifier.
func ValidateDBName(s string) string {
	if len(s) > 64 {
		return "Database name must be 64 characters or fewer"
	}
	if !mysqlIdentRe.MatchString(s) {
		return "Must start with a letter/underscore, only alphanumeric and underscores"
	}
	return ""
}

// ValidateDBUser checks if a string is a valid MySQL username.
func ValidateDBUser(s string) string {
	if len(s) > 32 {
		return "Username must be 32 characters or fewer"
	}
	if !mysqlIdentRe.MatchString(s) {
		return "Must start with a letter/underscore, only alphanumeric and underscores"
	}
	return ""
}

// ValidatePHPSize checks if a string is a valid PHP size value like "512M", "1024M", "2G".
func ValidatePHPSize(s string) string {
	s = strings.TrimSpace(s)
	if !phpSizeRe.MatchString(s) {
		return "Must be a number followed by M, G, or K (e.g., 512M, 1G)"
	}
	return ""
}

// ValidateDomain checks if a string looks like a valid domain name.
func ValidateDomain(s string) string {
	if len(s) > 253 {
		return "Domain name too long"
	}
	if !domainRe.MatchString(s) {
		return "Must be a valid domain name (e.g., nextcloud.example.com)"
	}
	return ""
}

// ValidatePath checks if a string is a valid absolute Unix path.
func ValidatePath(s string) string {
	if !strings.HasPrefix(s, "/") {
		return "Must be an absolute path starting with /"
	}
	if !pathRe.MatchString(s) {
		return "Path contains invalid characters"
	}
	return ""
}
