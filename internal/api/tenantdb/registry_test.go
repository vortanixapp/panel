package tenantdb

import "testing"

func TestReplaceDatabaseKeepsConnectionParameters(t *testing.T) {
	got := replaceDatabase("postgres://vortanix:secret@127.0.0.1:5432/vortanix?sslmode=disable", "vx_acme")

	want := "postgres://vortanix:secret@127.0.0.1:5432/vx_acme?sslmode=disable"
	if got != want {
		t.Fatalf("получено %q, ожидалось %q", got, want)
	}
}

func TestReplaceDatabaseWithoutQuery(t *testing.T) {
	got := replaceDatabase("postgres://user@host:5432/vortanix", "vx_acme")

	if got != "postgres://user@host:5432/vx_acme" {
		t.Fatalf("получено %q", got)
	}
}

func TestSlugMustBeSafeForCreateDatabase(t *testing.T) {
	bad := []string{
		`acme"; DROP DATABASE vortanix; --`,
		"acme database",
		"-acme",
		"acme-",
		"",
		"ACME",
	}
	for _, slug := range bad {
		if safeSlug.MatchString(slug) {
			t.Fatalf("слаг %q не должен доходить до CREATE DATABASE", slug)
		}
	}
	for _, slug := range []string{"acme", "panel-vortanix-app", "a1"} {
		if !safeSlug.MatchString(slug) {
			t.Fatalf("слаг %q должен приниматься", slug)
		}
	}
}
