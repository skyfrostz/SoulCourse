package handler

import "testing"

func TestBindDatabaseQuery(t *testing.T) {
	query := `UPDATE users SET password_hash = ? WHERE id = ? AND nickname <> '?'`
	if got := bindDatabaseQuery("sqlite", query); got != query {
		t.Fatalf("SQLite query changed: %s", got)
	}
	want := `UPDATE users SET password_hash = $1 WHERE id = $2 AND nickname <> '?'`
	if got := bindDatabaseQuery("postgres", query); got != want {
		t.Fatalf("PostgreSQL query mismatch: got %s want %s", got, want)
	}
}
