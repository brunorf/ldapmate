package main

import (
	"strings"
	"testing"
)

func TestGenerateRandomPassword(t *testing.T) {
	length := 12
	pass := generateRandomPassword(length)

	if len(pass) != length {
		t.Errorf("Expected password length %d, got %d", length, len(pass))
	}

	pass2 := generateRandomPassword(length)
	if pass == pass2 {
		t.Errorf("Expected random passwords to be different, but got twice: %s", pass)
	}

	if strings.ContainsAny(pass, " ") {
		t.Errorf("Password should not contain spaces: %s", pass)
	}
}
