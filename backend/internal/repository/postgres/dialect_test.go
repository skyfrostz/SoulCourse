package postgres

import "testing"

func TestRebindSkipsQuotedQuestionMarks(t *testing.T) {
	got := rebind(`SELECT '?' AS literal, "?" AS identifier FROM posts WHERE id = ? AND title = ?`)
	want := `SELECT '?' AS literal, "?" AS identifier FROM posts WHERE id = $1 AND title = $2`
	if got != want {
		t.Fatalf("rebind mismatch\nwant: %s\n got: %s", want, got)
	}
}
