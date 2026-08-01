package main

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
)

func TestConvertRowNormalizesJSONAndTimes(t *testing.T) {
	table := spec("posts", true,
		[]string{"id", "image_urls", "created_at", "deleted_at"},
		[]string{"image_urls"},
		[]string{"created_at", "deleted_at"},
		nil,
		nil,
	)

	values, err := convertRow(table, []any{int64(7), []byte(`["a.png"]`), "2026-07-31T12:34:56Z", ""})
	if err != nil {
		t.Fatalf("convertRow returned error: %v", err)
	}
	if values[1] != `["a.png"]` {
		t.Fatalf("expected JSON string, got %#v", values[1])
	}
	createdAt, ok := values[2].(time.Time)
	if !ok {
		t.Fatalf("expected created_at time, got %T", values[2])
	}
	if createdAt.Format(time.RFC3339) != "2026-07-31T12:34:56Z" {
		t.Fatalf("unexpected created_at %s", createdAt.Format(time.RFC3339))
	}
	if values[3] != nil {
		t.Fatalf("expected blank nullable time to become nil, got %#v", values[3])
	}
}

func TestConvertRowRejectsInvalidJSON(t *testing.T) {
	table := spec("admin_content_records", false, []string{"payload"}, []string{"payload"}, nil, nil, nil)
	if _, err := convertRow(table, []any{"not-json"}); err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

func TestInsertSQLCastsJSONB(t *testing.T) {
	table := spec("user_profiles", false, []string{"user_id", "choice_profile"}, []string{"choice_profile"}, nil, nil, nil)
	got := insertSQL(table)
	want := `INSERT INTO "user_profiles" ("user_id", "choice_profile") VALUES ($1, $2::jsonb)`
	if got != want {
		t.Fatalf("insertSQL mismatch\nwant: %s\n got: %s", want, got)
	}
}

func TestHashRowIsStable(t *testing.T) {
	hash := sha256.New()
	hashRow(hash, []string{"id", "created_at"}, []any{int64(1), time.Date(2026, 7, 31, 1, 2, 3, 0, time.UTC)})
	got := hex.EncodeToString(hash.Sum(nil))
	const want = "55582507a7158fefa16304980430119d5ebc04148b3c38d15a79ad32ba87d568"
	if got != want {
		t.Fatalf("hash mismatch: %s", got)
	}
}

func TestCanonicalJSONMatchesJSONBObjectOrdering(t *testing.T) {
	got, err := canonicalJSON(`{ "z": 1, "a": [true, null] }`)
	if err != nil {
		t.Fatalf("canonicalJSON returned error: %v", err)
	}
	const want = `{"a":[true,null],"z":1}`
	if got != want {
		t.Fatalf("canonical JSON mismatch: got %s want %s", got, want)
	}
}

func TestNormalizeTargetRowCanonicalizesJSONBytes(t *testing.T) {
	table := spec("posts", true, []string{"id", "tags"}, []string{"tags"}, nil, nil, nil)
	values, err := normalizeTargetRow(table, []any{int64(8), []byte(`[{"name":"化学"}]`)})
	if err != nil {
		t.Fatalf("normalizeTargetRow returned error: %v", err)
	}
	if values[1] != `[{"name":"化学"}]` {
		t.Fatalf("unexpected normalized JSON: %#v", values[1])
	}
}

func TestParseSQLiteTimeUsesPostgresMicrosecondPrecision(t *testing.T) {
	value, err := parseSQLiteTime("2026-07-05T15:33:40.9733389Z")
	if err != nil {
		t.Fatalf("parseSQLiteTime returned error: %v", err)
	}
	got := value.(time.Time).Format(time.RFC3339Nano)
	const want = "2026-07-05T15:33:40.973339Z"
	if got != want {
		t.Fatalf("timestamp precision mismatch: got %s want %s", got, want)
	}
}
