// Command backfill-post-tags classifies existing posts with the controlled
// topic taxonomy and adds the deterministic subject-combination tag.
// It is intentionally dry-run by default; production writes require -execute.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"
	"subject-choice-forum/backend/internal/service"
	"subject-choice-forum/backend/internal/storage"
)

type result struct {
	ID      int64    `json:"id"`
	OldTags []string `json:"oldTags"`
	NewTags []string `json:"newTags"`
	AITags  []string `json:"aiTags,omitempty"`
	AIError string   `json:"aiError,omitempty"`
	Changed bool     `json:"changed"`
}

type postRow struct {
	id                                           int64
	title, content, rawTags, track, rawElectives string
}

func main() {
	var execute bool
	var reportPath string
	var limit int
	flag.BoolVar(&execute, "execute", false, "apply tag updates; default is dry-run")
	flag.StringVar(&reportPath, "report", "", "write JSON audit report to this path")
	flag.IntVar(&limit, "limit", 0, "maximum posts to process; 0 means all")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fatal("load config", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	database, err := storage.NewDatabase(ctx, cfg)
	if err != nil {
		fatal("open database", err)
	}
	defer database.DB.Close()

	tagger := service.NewAIService(cfg)
	items, err := run(ctx, database.DB, database.Driver, tagger, execute, limit)
	if err != nil {
		fatal("backfill", err)
	}
	mode := "dry-run"
	if execute {
		mode = "execute"
	}
	changed := 0
	for _, item := range items {
		if item.Changed {
			changed++
		}
	}
	output := struct {
		Mode      string   `json:"mode"`
		Processed int      `json:"processed"`
		Changed   int      `json:"changed"`
		Items     []result `json:"items"`
	}{mode, len(items), changed, items}
	encoded, _ := json.MarshalIndent(output, "", "  ")
	if reportPath != "" {
		if err := os.WriteFile(reportPath, append(encoded, '\n'), 0600); err != nil {
			fatal("write report", err)
		}
	}
	fmt.Printf("backfill post tags %s: processed=%d changed=%d report=%s\n", mode, len(items), changed, reportPath)
}

func run(ctx context.Context, db *sql.DB, driver string, tagger interface {
	TagPost(context.Context, string, string) ([]string, error)
}, execute bool, limit int) ([]result, error) {
	query := `SELECT id, title, content, tags, track, electives FROM posts WHERE deleted_at IS NULL ORDER BY id`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	posts := make([]postRow, 0)
	for rows.Next() {
		if limit > 0 && len(posts) >= limit {
			break
		}
		var id int64
		var title, content, rawTags, track, rawElectives string
		if err := rows.Scan(&id, &title, &content, &rawTags, &track, &rawElectives); err != nil {
			return nil, err
		}
		posts = append(posts, postRow{id, title, content, rawTags, track, rawElectives})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	items := make([]result, 0, len(posts))
	for _, post := range posts {
		id := post.id
		oldTags := parseStrings(post.rawTags)
		newTags := append([]string(nil), oldTags...)
		aiTags, aiErr := tagger.TagPost(ctx, post.title, post.content)
		for _, tag := range aiTags {
			if domain.IsControlledTag(tag) {
				newTags = appendUnique(newTags, tag)
			}
		}
		electives := parseSubjects(post.rawElectives)
		if subjectTag, ok := domain.SubjectTagForChoice(domain.SubjectTrack(post.track), electives); ok {
			newTags = appendUnique(newTags, subjectTag)
		}
		item := result{ID: id, OldTags: oldTags, NewTags: newTags, AITags: aiTags, Changed: !same(oldTags, newTags)}
		if aiErr != nil {
			item.AIError = aiErr.Error()
		}
		if execute && item.Changed {
			if err := updateTags(ctx, db, driver, id, newTags); err != nil {
				return items, fmt.Errorf("post %d: %w", id, err)
			}
		}
		items = append(items, item)
	}
	return items, nil
}

func updateTags(ctx context.Context, db *sql.DB, driver string, id int64, tags []string) error {
	payload, err := json.Marshal(tags)
	if err != nil {
		return err
	}
	query := `UPDATE posts SET tags = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`
	if driver == "postgres" {
		query = `UPDATE posts SET tags = $1::jsonb, updated_at = $2 WHERE id = $3 AND deleted_at IS NULL`
	}
	_, err = db.ExecContext(ctx, query, string(payload), time.Now().UTC(), id)
	return err
}

func parseStrings(raw string) []string {
	var values []string
	_ = json.Unmarshal([]byte(raw), &values)
	return values
}
func parseSubjects(raw string) []domain.Subject {
	values := parseStrings(raw)
	out := make([]domain.Subject, 0, len(values))
	for _, v := range values {
		out = append(out, domain.Subject(v))
	}
	return out
}
func appendUnique(tags []string, value string) []string {
	for _, tag := range tags {
		if tag == value {
			return tags
		}
	}
	return append(tags, value)
}
func same(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func fatal(action string, err error) { fmt.Fprintf(os.Stderr, "%s: %v\n", action, err); os.Exit(1) }
