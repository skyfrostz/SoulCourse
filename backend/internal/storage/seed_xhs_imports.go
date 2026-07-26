package storage

import (
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const xhsImportMigration = "20260727-xhs-source-import-v2"

//go:embed xhs_data/*.json
var xhsSeedFiles embed.FS

type xhsSeedDetail struct {
	Province  string `json:"province"`
	FeedID    string `json:"feed_id"`
	XSecToken string `json:"xsec_token"`
	Note      struct {
		Title     string `json:"title"`
		Desc      string `json:"desc"`
		Time      int64  `json:"time"`
		ImageList []struct {
			URLDefault string `json:"urlDefault"`
		} `json:"imageList"`
		User struct {
			UserID   string `json:"userId"`
			Nickname string `json:"nickname"`
			Avatar   string `json:"avatar"`
		} `json:"user"`
		InteractInfo struct {
			LikedCount     string `json:"likedCount"`
			CommentCount   string `json:"commentCount"`
			CollectedCount string `json:"collectedCount"`
		} `json:"interactInfo"`
		TagList []struct {
			Name string `json:"name"`
		} `json:"tagList"`
	} `json:"note"`
}

var xhsHistoryPosts = map[string]bool{
	"688711230000000023007946": true,
	"6a37a6df0000000007029c90": true,
	"67b6a33200000000090140f3": true,
	"6a3279d6000000001702a5d9": true,
	"689c4f54000000001b01c700": true,
	"6a5cdd12000000000100c995": true,
	"6a2c0382000000001c026f59": true,
	"685d9ded000000001202080c": true,
	"6a39480e00000000060340b3": true,
	"6a2f5d76000000001c025f0a": true,
	"64be368e000000001700c607": true,
	"61d31c4c000000000102fd87": true,
	"66baaebe00000000090147c1": true,
	"688913d1000000000403ef1b": true,
	"62e926c90000000011011ad4": true,
	"68739ab6000000001c035204": true,
	"62cf792a000000000f00466a": true,
	"69c28da6000000002103a099": true,
	"6a58bee40000000013025e6c": true,
	"67a9deae0000000029011787": true,
	"672094a9000000001a01dccd": true,
	"6a315c83000000001603e431": true,
	"6a2928bc00000000170295cd": true,
	"66ebc123000000000c0184ec": true,
	"68b1ba5c000000001c00feb6": true,
}

var xhsElectiveOverrides = map[string][]string{
	"659fc06a000000001a02bb3c": {"biology", "geography"},
	"66545ea20000000005004153": {"biology", "geography"},
	"65df28bb0000000003035eb0": {"politics", "geography"},
	"67120549000000002100821f": {"politics", "geography"},
	"67a9deae0000000029011787": {"chemistry", "biology"},
	"6a315c83000000001603e431": {"biology", "politics"},
	"6a2928bc00000000170295cd": {"biology", "geography"},
	"66ebc123000000000c0184ec": {"geography", "politics"},
	"68b1ba5c000000001c00feb6": {"politics", "biology"},
}

func seedXHSImports(db *sql.DB) error {
	var applied int
	if err := db.QueryRow(`SELECT COUNT(*) FROM app_migrations WHERE name = ?`, xhsImportMigration).Scan(&applied); err != nil {
		return err
	}
	if applied > 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	if _, err := tx.Exec(`
		UPDATE posts
		SET author_name = (SELECT source_author FROM content_sources WHERE post_id = posts.id),
			user_id = NULL,
			author_role = 'student',
			updated_at = ?
		WHERE id IN (SELECT post_id FROM content_sources)
	`, now.Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("update sourced post attribution: %w", err)
	}
	if err := syncExistingSourceAdminRecords(tx); err != nil {
		return err
	}

	entries, err := fs.ReadDir(xhsSeedFiles, "xhs_data")
	if err != nil {
		return err
	}
	for index, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := xhsSeedFiles.ReadFile("xhs_data/" + entry.Name())
		if err != nil {
			return err
		}
		var detail xhsSeedDetail
		if err := json.Unmarshal(data, &detail); err != nil {
			return fmt.Errorf("decode %s: %w", entry.Name(), err)
		}
		if err := insertXHSSeed(tx, detail, index, now); err != nil {
			return err
		}
	}

	for _, user := range guangdongUsers {
		if _, err := tx.Exec(`
			UPDATE users SET deleted_at = ?, updated_at = ?
			WHERE lower(email) = lower(?)
			  AND NOT EXISTS (SELECT 1 FROM posts WHERE posts.user_id = users.id AND posts.deleted_at IS NULL)
		`, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), user.email); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(`INSERT INTO app_migrations (name, applied_at) VALUES (?, ?)`, xhsImportMigration, now.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return tx.Commit()
}

func syncExistingSourceAdminRecords(tx *sql.Tx) error {
	rows, err := tx.Query(`
		SELECT source_note_id, source_url, source_title, source_author, source_avatar_url
		FROM content_sources
	`)
	if err != nil {
		return err
	}
	type sourceRecord struct {
		noteID, sourceURL, title, author, avatar string
	}
	records := make([]sourceRecord, 0)
	for rows.Next() {
		var record sourceRecord
		if err := rows.Scan(&record.noteID, &record.sourceURL, &record.title, &record.author, &record.avatar); err != nil {
			rows.Close()
			return err
		}
		records = append(records, record)
	}
	if err := rows.Close(); err != nil {
		return err
	}

	for _, record := range records {
		var payloadRaw string
		if err := tx.QueryRow(`SELECT payload FROM admin_content_records WHERE id = ?`, "xhs-"+record.noteID).Scan(&payloadRaw); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return err
		}
		payload := map[string]any{}
		_ = json.Unmarshal([]byte(payloadRaw), &payload)
		payload["sourcePlatform"] = "xiaohongshu"
		payload["sourceUrl"] = record.sourceURL
		payload["sourceTitle"] = record.title
		payload["sourceAuthor"] = record.author
		payload["sourceAvatarUrl"] = record.avatar
		if _, err := tx.Exec(`UPDATE admin_content_records SET owner = ?, payload = ? WHERE id = ?`,
			record.author, jsonString(payload), "xhs-"+record.noteID); err != nil {
			return err
		}
	}
	return nil
}

func insertXHSSeed(tx *sql.Tx, detail xhsSeedDetail, index int, now time.Time) error {
	var exists int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM content_sources WHERE source_note_id = ?`, detail.FeedID).Scan(&exists); err != nil {
		return err
	}
	if exists > 0 {
		return nil
	}

	track := "physics"
	electives := []string{"chemistry", "biology"}
	if xhsHistoryPosts[detail.FeedID] {
		track = "history"
		electives = []string{"politics", "geography"}
	}
	if override := xhsElectiveOverrides[detail.FeedID]; len(override) == 2 {
		electives = override
	}

	province := xhsProvince(detail.Province)
	category := xhsCategory(detail.Note.Title)
	grade := "高一"
	if detail.Province != "guangdong" || strings.Contains(detail.Note.Title, "志愿") || strings.Contains(detail.Note.Title, "投档") {
		grade = "毕业生"
	}

	images := make([]string, 0, min(len(detail.Note.ImageList), 9))
	for imageIndex := 1; imageIndex <= len(detail.Note.ImageList) && imageIndex <= 9; imageIndex++ {
		images = append(images, fmt.Sprintf("/content/xhs/%s/%d.webp", detail.FeedID, imageIndex))
	}
	tags := xhsTags(detail)
	createdAt := now.Add(-time.Duration(index) * time.Hour)
	if detail.Note.Time > 0 {
		createdAt = time.UnixMilli(detail.Note.Time).UTC()
	}

	result, err := tx.Exec(`
		INSERT INTO posts
			(user_id, author_name, author_role, title, content, image_urls, tags, track, electives,
			 category, grade, province, likes_count, comments_count, favorites_count, created_at, updated_at)
		VALUES (NULL, ?, 'student', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, detail.Note.User.Nickname, detail.Note.Title, detail.Note.Desc, jsonString(images), jsonString(tags),
		track, jsonString(electives), category, grade, province,
		parseXHSCount(detail.Note.InteractInfo.LikedCount), parseXHSCount(detail.Note.InteractInfo.CommentCount),
		parseXHSCount(detail.Note.InteractInfo.CollectedCount), createdAt.Format(time.RFC3339Nano), createdAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("insert sourced post %s: %w", detail.FeedID, err)
	}
	postID, err := result.LastInsertId()
	if err != nil {
		return err
	}

	sourceURL := "https://www.xiaohongshu.com/explore/" + detail.FeedID
	if detail.XSecToken != "" {
		sourceURL += "?xsec_token=" + url.QueryEscape(detail.XSecToken) + "&xsec_source=pc_search"
	}
	avatarURL := fmt.Sprintf("/content/xhs/%s/avatar.jpg", detail.FeedID)
	if _, err := tx.Exec(`
		INSERT INTO content_sources
			(post_id, source_platform, source_url, source_note_id, source_title, source_author, source_avatar_url,
			 source_likes, source_comments, source_favorites, source_format, transformation_note, captured_at)
		VALUES (?, 'xiaohongshu', ?, ?, ?, ?, ?, ?, ?, ?, '图文', ?, ?)
	`, postID, sourceURL, detail.FeedID, detail.Note.Title, detail.Note.User.Nickname, avatarURL,
		parseXHSCount(detail.Note.InteractInfo.LikedCount), parseXHSCount(detail.Note.InteractInfo.CommentCount),
		parseXHSCount(detail.Note.InteractInfo.CollectedCount),
		"小红书公开笔记原文及图片迁移；站内明确标注来源，来源作者不具备站内登录身份。", now.Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("insert source %s: %w", detail.FeedID, err)
	}

	payload := map[string]any{
		"postId": postID, "content": detail.Note.Desc, "track": track, "electives": electives,
		"category": category, "grade": grade, "province": province, "imageUrls": images,
		"sourcePlatform": "xiaohongshu", "sourceUrl": sourceURL, "sourceTitle": detail.Note.Title,
		"sourceAuthor": detail.Note.User.Nickname, "sourceAvatarUrl": avatarURL, "sourceUserId": detail.Note.User.UserID,
	}
	if _, err := tx.Exec(`
		INSERT INTO admin_content_records
			(id, module, title, content_type, status, scope, owner, tags, summary, url,
			 priority, sort_order, payload, created_at, updated_at)
		VALUES (?, 'posts', ?, ?, '已上架', ?, ?, ?, ?, ?, '常规', ?, ?, ?, ?)
	`, "xhs-import-"+detail.FeedID, detail.Note.Title, seededPostType(category), province,
		detail.Note.User.Nickname, jsonString(tags), summarizeSeedContent(detail.Note.Desc),
		fmt.Sprintf("/posts/%d", postID), 500+index, jsonString(payload),
		createdAt.Format(time.RFC3339Nano), createdAt.Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("insert admin source %s: %w", detail.FeedID, err)
	}
	return nil
}

func xhsProvince(value string) string {
	return map[string]string{
		"guangdong": "广东", "hunan": "湖南", "fujian": "福建",
		"guangxi": "广西", "zhejiang": "浙江", "jiangsu": "江苏",
	}[value]
}

func xhsCategory(title string) string {
	for _, keyword := range []string{"分数", "投档", "排名", "比例", "覆盖率", "赋分", "要求"} {
		if strings.Contains(title, keyword) {
			return "data"
		}
	}
	return "experience"
}

func xhsTags(detail xhsSeedDetail) []string {
	tags := make([]string, 0, 8)
	seen := map[string]bool{}
	for _, tag := range detail.Note.TagList {
		name := strings.TrimSpace(tag.Name)
		if name == "" || seen[name] {
			continue
		}
		tags = append(tags, name)
		seen[name] = true
		if len(tags) == 7 {
			break
		}
	}
	return append(tags, "小红书来源")
}

func parseXHSCount(value string) int {
	count, _ := strconv.Atoi(strings.TrimSpace(value))
	return count
}

func jsonString(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}
