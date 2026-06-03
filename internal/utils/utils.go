// Package config loads runtime settings (port, database DSN, secrets) from the
// environment. This is the only place that reads env vars.
package utils

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}
