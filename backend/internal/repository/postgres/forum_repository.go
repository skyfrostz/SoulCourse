package postgres

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"subject-choice-forum/backend/internal/domain"
)

type ForumRepository struct {
	db *sql.DB
}

func NewForumRepository(db *sql.DB) *ForumRepository {
	return &ForumRepository{db: db}
}

func (r *ForumRepository) ListPosts(ctx context.Context, viewerID *int64, filter domain.FeedFilter) (domain.PostPage, error) {
	filter.Sort = normalizePostSort(filter.Sort)
	if filter.Cursor != "" {
		var err error
		switch filter.Sort {
		case domain.SortLatest:
			_, _, err = parsePostCursor(filter.Cursor)
		case domain.SortHot, domain.SortRecommended:
			_, err = parseRankedPostCursor(filter.Cursor, filter.Sort)
		}
		if err != nil {
			return domain.PostPage{}, fmt.Errorf("invalid post cursor: %w", err)
		}
	}
	query, args := buildPostListQuery(filter)
	rows, err := r.query(ctx, query, args...)
	if err != nil {
		return domain.PostPage{}, err
	}
	defer rows.Close()

	posts := make([]domain.Post, 0, filter.Limit+1)
	for rows.Next() {
		post, err := scanPost(rows)
		if err != nil {
			return domain.PostPage{}, err
		}
		posts = append(posts, post)
	}
	if err := rows.Err(); err != nil {
		return domain.PostPage{}, err
	}

	liked, favorited, followed, err := r.viewerState(ctx, viewerID)
	if err != nil {
		return domain.PostPage{}, err
	}

	page := domain.PostPage{Items: posts}
	if filter.Limit > 0 && len(posts) > filter.Limit {
		page.HasMore = true
		page.Items = posts[:filter.Limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = postPageCursor(filter.Sort, last)
	}
	for i := range page.Items {
		post := &page.Items[i]
		post.ViewerLiked = liked[post.ID]
		post.ViewerFavorited = favorited[post.ID]
		post.ViewerFollowing = post.SourcePlatform == "" && followed[post.AuthorName]
	}
	return page, nil
}

func normalizePostSort(sortOrder domain.FeedSort) domain.FeedSort {
	switch sortOrder {
	case domain.SortLatest, domain.SortHot, domain.SortRecommended:
		return sortOrder
	default:
		return domain.SortRecommended
	}
}

func buildPostListQuery(filter domain.FeedFilter) (string, []any) {
	query := `
		SELECT p.id, p.user_id, p.author_name, p.author_role, p.title, p.content, p.image_urls, p.tags, p.track, p.electives,
		       p.category, p.grade, p.province, p.likes_count, p.comments_count, p.favorites_count, p.created_at, p.updated_at,
		       COALESCE(cs.source_platform, ''), COALESCE(cs.source_url, ''), COALESCE(cs.source_title, ''),
		       COALESCE(cs.source_author, ''), COALESCE(cs.source_avatar_url, '')
		FROM posts p
		LEFT JOIN content_sources cs ON cs.post_id = p.id
		WHERE p.deleted_at IS NULL`
	args := make([]any, 0, 8)

	if filter.Track != "" {
		query += " AND p.track = ?"
		args = append(args, string(filter.Track))
	}
	if filter.Category != "" {
		query += " AND p.category = ?"
		args = append(args, string(filter.Category))
	}
	if province := strings.TrimSpace(filter.Province); province != "" {
		query += " AND p.province = ?"
		args = append(args, province)
	}
	if tag := strings.TrimSpace(filter.Tag); tag != "" {
		query += ` AND p.tags @> jsonb_build_array(?::text)`
		args = append(args, tag)
	}
	if filter.UserID != nil {
		query += " AND p.user_id = ?"
		args = append(args, *filter.UserID)
	}
	if filter.Cursor != "" {
		switch filter.Sort {
		case domain.SortLatest:
			if cursorTime, cursorID, err := parsePostCursor(filter.Cursor); err == nil {
				query += " AND (p.created_at < ? OR (p.created_at = ? AND p.id < ?))"
				args = append(args, cursorTime, cursorTime, cursorID)
			}
		case domain.SortHot, domain.SortRecommended:
			if cursor, err := parseRankedPostCursor(filter.Cursor, filter.Sort); err == nil {
				rankExpression := postRankExpression(filter.Sort)
				query += " AND (" + rankExpression + " < ? OR (" + rankExpression + " = ? AND (p.created_at < ? OR (p.created_at = ? AND p.id < ?))))"
				args = append(args, cursor.Rank, cursor.Rank, cursor.CreatedAt, cursor.CreatedAt, cursor.ID)
			}
		}
	}

	subjects := make([]domain.Subject, 0, len(filter.Subjects)+1)
	subjects = append(subjects, filter.Subjects...)
	if filter.Subject != "" {
		subjects = append(subjects, filter.Subject)
	}
	for _, subject := range subjects {
		if subject == "" {
			continue
		}
		query += ` AND p.electives @> jsonb_build_array(?::text)`
		args = append(args, string(subject))
	}

	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		query += ` AND p.search_vector @@ plainto_tsquery('simple', ?::text)`
		args = append(args, keyword)
	}
	switch filter.Sort {
	case domain.SortLatest:
		query += " ORDER BY p.created_at DESC, p.id DESC"
	case domain.SortHot:
		query += " ORDER BY " + postRankExpression(domain.SortHot) + " DESC, p.created_at DESC, p.id DESC"
	default:
		query += " ORDER BY " + postRankExpression(domain.SortRecommended) + " DESC, p.created_at DESC, p.id DESC"
	}

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit+1)
		if filter.Cursor == "" && filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}
	return query, args
}

func postRankExpression(sortOrder domain.FeedSort) string {
	if sortOrder == domain.SortHot {
		return "(p.likes_count + p.comments_count * 4)"
	}
	return `(CASE WHEN p.title LIKE '%选科%' OR p.tags::text LIKE '%选科%' THEN 1500 ELSE 0 END
		+ LEAST(p.likes_count, 300) * 0.8 + LEAST(p.comments_count, 80) * 4 + LEAST(p.favorites_count, 120) * 3
		+ CASE WHEN p.author_role IN ('teacher', 'counselor') THEN 45 ELSE 0 END
		+ CASE WHEN p.likes_count < 150 THEN 65 ELSE 0 END)`
}

type rankedPostCursor struct {
	Sort      domain.FeedSort `json:"s"`
	Rank      float64         `json:"r"`
	CreatedAt string          `json:"t"`
	ID        int64           `json:"i"`
}

func postPageCursor(sortOrder domain.FeedSort, post domain.Post) string {
	if sortOrder == domain.SortLatest {
		return postCursor(post.CreatedAt, post.ID)
	}
	cursor := rankedPostCursor{
		Sort:      sortOrder,
		Rank:      postRank(sortOrder, post),
		CreatedAt: post.CreatedAt.UTC().Format(time.RFC3339Nano),
		ID:        post.ID,
	}
	payload, _ := json.Marshal(cursor)
	return "r1." + base64.RawURLEncoding.EncodeToString(payload)
}

func parseRankedPostCursor(value string, sortOrder domain.FeedSort) (rankedPostCursor, error) {
	var cursor rankedPostCursor
	if !strings.HasPrefix(value, "r1.") {
		return cursor, fmt.Errorf("invalid ranked post cursor")
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, "r1."))
	if err != nil || json.Unmarshal(payload, &cursor) != nil || cursor.Sort != sortOrder || cursor.ID <= 0 {
		return rankedPostCursor{}, fmt.Errorf("invalid ranked post cursor")
	}
	if _, err := time.Parse(time.RFC3339Nano, cursor.CreatedAt); err != nil {
		return rankedPostCursor{}, fmt.Errorf("invalid ranked post cursor")
	}
	return cursor, nil
}

func postRank(sortOrder domain.FeedSort, post domain.Post) float64 {
	if sortOrder == domain.SortHot {
		return float64(post.LikesCount + post.CommentsCount*4)
	}
	rank := float64(min(post.LikesCount, 300))*0.8 + float64(min(post.CommentsCount, 80)*4+min(post.FavoritesCount, 120)*3)
	if strings.Contains(post.Title, "选科") || slicesContainSubstring(post.Tags, "选科") {
		rank += 1500
	}
	if post.AuthorRole == "teacher" || post.AuthorRole == "counselor" {
		rank += 45
	}
	if post.LikesCount < 150 {
		rank += 65
	}
	return rank
}

func slicesContainSubstring(values []string, substring string) bool {
	for _, value := range values {
		if strings.Contains(value, substring) {
			return true
		}
	}
	return false
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "%", `\%`)
	return strings.ReplaceAll(value, "_", `\_`)
}

func (r *ForumRepository) GetPost(ctx context.Context, viewerID *int64, id int64) (domain.Post, []domain.Comment, error) {
	post, err := r.fetchPostByID(ctx, id)
	if err != nil {
		return domain.Post{}, nil, err
	}
	liked, favorited, followed, err := r.viewerState(ctx, viewerID)
	if err != nil {
		return domain.Post{}, nil, err
	}
	post.ViewerLiked = liked[post.ID]
	post.ViewerFavorited = favorited[post.ID]
	post.ViewerFollowing = post.SourcePlatform == "" && followed[post.AuthorName]

	rows, err := r.query(ctx, `
		SELECT id, post_id, user_id, author, role, content, created_at
		FROM comments
		WHERE post_id = ? AND deleted_at IS NULL
		ORDER BY created_at ASC
	`, id)
	if err != nil {
		return domain.Post{}, nil, err
	}
	defer rows.Close()

	comments := make([]domain.Comment, 0)
	for rows.Next() {
		comment, err := scanComment(rows)
		if err != nil {
			return domain.Post{}, nil, err
		}
		comments = append(comments, comment)
	}
	return post, comments, rows.Err()
}

func (r *ForumRepository) CreatePost(ctx context.Context, user domain.User, input domain.CreatePostInput) (domain.Post, error) {
	now := nowString()
	var postID int64
	err := r.queryRow(ctx, `
		INSERT INTO posts (user_id, author_name, author_role, title, content, image_urls, tags, track, electives, category, grade, province, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id
	`,
		user.ID,
		user.Nickname,
		user.Role,
		input.Title,
		input.Content,
		mustJSON(input.ImageURLs),
		mustJSON(input.Tags),
		string(input.Track),
		mustJSON(subjectStrings(input.Electives)),
		string(input.Category),
		user.Grade,
		user.Province,
		now,
		now,
	).Scan(&postID)
	if err != nil {
		return domain.Post{}, err
	}

	payload := map[string]any{
		"postId":          fmt.Sprintf("%d", postID),
		"content":         input.Content,
		"track":           input.Track,
		"electives":       input.Electives,
		"category":        input.Category,
		"grade":           user.Grade,
		"province":        user.Province,
		"imageUrls":       input.ImageURLs,
		"createdByUserId": fmt.Sprintf("%d", user.ID),
	}
	_, err = r.exec(ctx, `
		INSERT INTO admin_content_records
			(id, module, title, content_type, status, scope, owner, tags, summary, url, priority, sort_order, payload, created_at, updated_at)
		VALUES (?, 'posts', ?, ?, '已上架', ?, ?, ?, ?, ?, '常规', 0, ?, ?, ?)
		ON CONFLICT (id) DO NOTHING
	`,
		fmt.Sprintf("post-user-%d", postID),
		input.Title,
		postContentType(input.Category),
		user.Province,
		user.Nickname,
		mustJSON(input.Tags),
		input.Content,
		fmt.Sprintf("/posts/%d", postID),
		mustJSON(payload),
		now,
		now,
	)
	if err != nil {
		return domain.Post{}, err
	}

	return r.fetchPostByID(ctx, postID)
}

func (r *ForumRepository) UpdatePost(ctx context.Context, userID int64, postID int64, input domain.UpdatePostInput) (domain.Post, error) {
	now := nowString()
	transaction, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Post{}, err
	}
	defer transaction.Rollback()

	result, err := execTx(ctx, transaction, `
		UPDATE posts
		SET title = ?, content = ?, tags = ?, track = ?, electives = ?, category = ?, updated_at = ?
		WHERE id = ? AND user_id = ? AND deleted_at IS NULL
	`,
		input.Title,
		input.Content,
		mustJSON(input.Tags),
		string(input.Track),
		mustJSON(subjectStrings(input.Electives)),
		string(input.Category),
		now,
		postID,
		userID,
	)
	if err != nil {
		return domain.Post{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return domain.Post{}, err
	}
	if affected == 0 {
		return domain.Post{}, sql.ErrNoRows
	}

	var grade string
	var province string
	var imageURLsRaw string
	if err := queryRowTx(ctx, transaction, `
		SELECT grade, province, image_urls FROM posts WHERE id = ?
	`, postID).Scan(&grade, &province, &imageURLsRaw); err != nil {
		return domain.Post{}, err
	}
	payload := map[string]any{
		"postId":          fmt.Sprintf("%d", postID),
		"content":         input.Content,
		"track":           input.Track,
		"electives":       input.Electives,
		"category":        input.Category,
		"grade":           grade,
		"province":        province,
		"imageUrls":       parseStringSlice(imageURLsRaw),
		"createdByUserId": fmt.Sprintf("%d", userID),
		"editedByUserId":  fmt.Sprintf("%d", userID),
	}
	if _, err := execTx(ctx, transaction, `
		UPDATE admin_content_records
		SET title = ?, content_type = ?, tags = ?, summary = ?, payload = ?, updated_at = ?
		WHERE id = ? AND module = 'posts' AND deleted_at IS NULL
	`,
		input.Title,
		postContentType(input.Category),
		mustJSON(input.Tags),
		input.Content,
		mustJSON(payload),
		now,
		fmt.Sprintf("post-user-%d", postID),
	); err != nil {
		return domain.Post{}, err
	}
	if err := transaction.Commit(); err != nil {
		return domain.Post{}, err
	}
	return r.fetchPostByID(ctx, postID)
}

func (r *ForumRepository) DeletePost(ctx context.Context, userID int64, postID int64) error {
	now := nowString()
	transaction, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer transaction.Rollback()

	result, err := execTx(ctx, transaction, `
		UPDATE posts
		SET deleted_at = COALESCE(deleted_at, ?), updated_at = ?
		WHERE id = ? AND user_id = ? AND deleted_at IS NULL
	`, now, now, postID, userID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	if _, err := execTx(ctx, transaction, `
		UPDATE admin_content_records
		SET deleted_at = COALESCE(deleted_at, ?), status = '用户已删除', updated_at = ?
		WHERE id = ? AND module = 'posts'
	`, now, now, fmt.Sprintf("post-user-%d", postID)); err != nil {
		return err
	}
	return transaction.Commit()
}

func (r *ForumRepository) CreateComment(ctx context.Context, user domain.User, postID int64, input domain.CreateCommentInput) (domain.Comment, error) {
	post, err := r.fetchPostByID(ctx, postID)
	if err != nil {
		return domain.Comment{}, err
	}
	now := nowString()
	var commentID int64
	err = r.queryRow(ctx, `
		INSERT INTO comments (post_id, user_id, author, role, content, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		RETURNING id
	`, postID, user.ID, user.Nickname, user.Role, input.Content, now).Scan(&commentID)
	if err != nil {
		return domain.Comment{}, err
	}
	if _, err := r.exec(ctx, `UPDATE posts SET comments_count = comments_count + 1, updated_at = ? WHERE id = ?`, now, postID); err != nil {
		return domain.Comment{}, err
	}
	if post.UserID != nil && *post.UserID != user.ID {
		_ = r.createNotification(ctx, *post.UserID, &user.ID, "comment", user.Nickname+" 评论了你的帖子", truncateText(input.Content, 90), fmt.Sprintf("/posts/%d", postID))
	}
	return scanComment(r.queryRow(ctx, `
		SELECT id, post_id, user_id, author, role, content, created_at
		FROM comments
		WHERE id = ?
	`, commentID))
}

func (r *ForumRepository) ReportPost(ctx context.Context, user domain.User, postID int64, input domain.ReportPostInput) (domain.ContentReport, error) {
	if _, err := r.fetchPostByID(ctx, postID); err != nil {
		return domain.ContentReport{}, err
	}
	now := nowString()
	var reportID int64
	err := r.queryRow(ctx, `
		INSERT INTO content_reports (reporter_user_id, target_type, target_id, reason, detail, created_at, updated_at)
		VALUES (?, 'post', ?, ?, ?, ?, ?)
		ON CONFLICT(reporter_user_id, target_type, target_id) DO UPDATE SET
			reason = excluded.reason,
			detail = excluded.detail,
			status = 'open',
			resolution_note = '',
			resolved_at = NULL,
			updated_at = excluded.updated_at
		RETURNING id
	`, user.ID, postID, input.Reason, input.Detail, now, now).Scan(&reportID)
	if err != nil {
		return domain.ContentReport{}, err
	}
	return r.scanContentReportByID(ctx, reportID)
}

func (r *ForumRepository) ListInsights(ctx context.Context) ([]domain.SubjectInsight, error) {
	rows, err := r.query(ctx, `
		SELECT id, combination, trend, heat, match_rate, advice, details,
		       metric_type, unit, province, data_year, source_name, source_url, scope,
		       sample_size, captured_at, methodology, updated_at
		FROM subject_insights
		ORDER BY heat DESC, id ASC
		LIMIT 8
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.SubjectInsight, 0)
	for rows.Next() {
		item, err := scanInsight(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *ForumRepository) GetInsight(ctx context.Context, id int64) (domain.SubjectInsight, error) {
	return scanInsight(r.queryRow(ctx, `
		SELECT id, combination, trend, heat, match_rate, advice, details,
		       metric_type, unit, province, data_year, source_name, source_url, scope,
		       sample_size, captured_at, methodology, updated_at
		FROM subject_insights
		WHERE id = ?
	`, id))
}

func (r *ForumRepository) ListTopics(ctx context.Context) ([]domain.Topic, error) {
	rows, err := r.query(ctx, `
		SELECT t.id, t.slug, t.topic_tag, t.title, t.summary, t.views_count,
		       (
			   SELECT COUNT(*)
			   FROM posts p
			   WHERE p.deleted_at IS NULL
			     AND t.topic_tag <> ''
			     AND p.tags @> jsonb_build_array(t.topic_tag)
		       ),
		       t.created_at
		FROM topics t
		ORDER BY t.views_count DESC, t.id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.Topic, 0)
	for rows.Next() {
		item, err := scanTopic(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *ForumRepository) GetTopic(ctx context.Context, viewerID *int64, slug string) (domain.TopicDetail, error) {
	if _, err := r.exec(ctx, `UPDATE topics SET views_count = views_count + 1 WHERE slug = ?`, slug); err != nil {
		return domain.TopicDetail{}, err
	}
	topic, err := scanTopic(r.queryRow(ctx, `
		SELECT t.id, t.slug, t.topic_tag, t.title, t.summary, t.views_count,
		       (
			   SELECT COUNT(*)
			   FROM posts p
			   WHERE p.deleted_at IS NULL
			     AND t.topic_tag <> ''
			     AND p.tags @> jsonb_build_array(t.topic_tag)
		       ),
		       t.created_at
		FROM topics t
		WHERE t.slug = ?
	`, slug))
	if err != nil {
		return domain.TopicDetail{}, err
	}
	posts := make([]domain.Post, 0)
	if topic.TopicTag != "" {
		postPage, pageErr := r.ListPosts(ctx, viewerID, domain.FeedFilter{
			Tag:  topic.TopicTag,
			Sort: domain.SortLatest,
		})
		if pageErr != nil {
			return domain.TopicDetail{}, pageErr
		}
		posts = postPage.Items
	}
	return domain.TopicDetail{Topic: topic, Posts: posts}, nil
}

func (r *ForumRepository) CreateUser(ctx context.Context, input domain.RegisterInput, passwordHash string) (domain.User, error) {
	now := nowString()
	var id int64
	err := r.queryRow(ctx, `
		INSERT INTO users (email, password_hash, nickname, role, province, grade, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id
	`, input.Email, passwordHash, input.Nickname, input.Role, input.Province, input.Grade, now, now).Scan(&id)
	if err != nil {
		return domain.User{}, err
	}
	nowProfile := nowString()
	if _, err := r.exec(ctx, `
		INSERT INTO user_profiles (user_id, bio, choice_profile, created_at, updated_at)
		VALUES (?, '', ?, ?, ?)
	`, id, mustJSON(defaultChoiceProfile()), nowProfile, nowProfile); err != nil {
		return domain.User{}, err
	}
	if err := r.createNotification(ctx, id, nil, "profile", "完善你的选科画像", "补充 MBTI、目标专业和学科稳定性，让建议更贴近你。", "/settings"); err != nil {
		return domain.User{}, err
	}
	return scanUser(r.queryRow(ctx, `
		SELECT id, email, nickname, role, province, grade, created_at
		FROM users
		WHERE id = ?
	`, id))
}

func (r *ForumRepository) GetUserByEmail(ctx context.Context, email string) (domain.User, string, error) {
	var user domain.User
	var createdAt string
	var passwordHash string
	err := r.queryRow(ctx, `
		SELECT id, email, nickname, role, province, grade, created_at, password_hash
		FROM users
		WHERE lower(email) = lower(?) AND deleted_at IS NULL AND banned_at IS NULL
	`, email).Scan(&user.ID, &user.Email, &user.Nickname, &user.Role, &user.Province, &user.Grade, &createdAt, &passwordHash)
	if err != nil {
		return domain.User{}, "", err
	}
	user.PublicID = formatUserPublicID(user.ID)
	user.CreatedAt = parseTime(createdAt)
	return user, passwordHash, nil
}

func (r *ForumRepository) GetUserByID(ctx context.Context, id int64) (domain.User, error) {
	return scanUser(r.queryRow(ctx, `
		SELECT id, email, nickname, role, province, grade, created_at
		FROM users
		WHERE id = ? AND deleted_at IS NULL
	`, id))
}

func (r *ForumRepository) GetUserPasswordHashByID(ctx context.Context, id int64) (string, error) {
	var passwordHash string
	err := r.queryRow(ctx, `
		SELECT password_hash
		FROM users
		WHERE id = ? AND deleted_at IS NULL AND is_shadow = false
	`, id).Scan(&passwordHash)
	return passwordHash, err
}

func (r *ForumRepository) UpdateUserPasswordByEmail(ctx context.Context, email string, passwordHash string, now time.Time) (int64, error) {
	result, err := r.exec(ctx, `
		UPDATE users
		SET password_hash = ?, updated_at = ?
		WHERE lower(email) = lower(?) AND deleted_at IS NULL AND is_shadow = false
	`, passwordHash, now.UTC().Format(time.RFC3339Nano), email)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if affected == 0 {
		return 0, sql.ErrNoRows
	}
	var userID int64
	if err := r.queryRow(ctx, `
		SELECT id FROM users
		WHERE lower(email) = lower(?) AND deleted_at IS NULL AND is_shadow = false
	`, email).Scan(&userID); err != nil {
		return 0, err
	}
	return userID, nil
}

func (r *ForumRepository) DeleteUserAccount(ctx context.Context, userID int64, now time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	timestamp := now.UTC().Format(time.RFC3339Nano)
	result, err := execTx(ctx, tx, `
		UPDATE users
		SET email = NULL,
		    password_hash = NULL,
		    nickname = ?,
		    deleted_at = ?,
		    updated_at = ?
		WHERE id = ? AND deleted_at IS NULL AND is_shadow = false
	`, fmt.Sprintf("已注销用户-%d", userID), timestamp, timestamp, userID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	if _, err := execTx(ctx, tx, `
		UPDATE auth_sessions
		SET revoked_at = ?
		WHERE user_id = ? AND revoked_at IS NULL
	`, timestamp, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ForumRepository) CreateImageUpload(ctx context.Context, record domain.ImageUploadRecord) error {
	_, err := r.exec(ctx, `
		INSERT INTO upload_assets (id, user_id, asset_key, file_name, content_type, ext, size_bytes, width, height, status, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, record.ID, record.UserID, record.AssetKey, record.FileName, record.ContentType, record.Ext, record.SizeBytes, record.Width, record.Height, record.Status, record.CreatedAt.UTC().Format(time.RFC3339Nano), record.ExpiresAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (r *ForumRepository) GetImageUpload(ctx context.Context, userID int64, id string) (domain.ImageUploadRecord, error) {
	return scanImageUpload(r.queryRow(ctx, `
		SELECT id, user_id, asset_key, file_name, content_type, ext, size_bytes, width, height, status, created_at, expires_at, completed_at
		FROM upload_assets
		WHERE id = ? AND user_id = ?
	`, id, userID))
}

func (r *ForumRepository) ListExpiredPendingImageUploads(ctx context.Context, now time.Time, limit int) ([]domain.ImageUploadRecord, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := r.query(ctx, `
		SELECT id, user_id, asset_key, file_name, content_type, ext, size_bytes, width, height, status, created_at, expires_at, completed_at
		FROM upload_assets
		WHERE status = 'pending' AND expires_at < ?
		ORDER BY expires_at ASC
		LIMIT ?
	`, now.UTC().Format(time.RFC3339Nano), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := make([]domain.ImageUploadRecord, 0)
	for rows.Next() {
		record, err := scanImageUpload(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (r *ForumRepository) MarkImageUploadsExpired(ctx context.Context, ids []string, now time.Time) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	placeholders := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids)+1)
	args = append(args, now.UTC().Format(time.RFC3339Nano))
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		placeholders = append(placeholders, "?")
		args = append(args, trimmed)
	}
	if len(placeholders) == 0 {
		return 0, nil
	}
	result, err := r.exec(ctx, `
		UPDATE upload_assets
		SET status = 'expired', completed_at = ?
		WHERE status = 'pending' AND id IN (`+strings.Join(placeholders, ",")+`)
	`, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *ForumRepository) CompleteImageUpload(ctx context.Context, userID int64, id string, sizeBytes int64, contentType string, width int, height int, now time.Time) (domain.ImageUploadRecord, error) {
	result, err := r.exec(ctx, `
		UPDATE upload_assets
		SET size_bytes = ?, content_type = ?, width = ?, height = ?, status = 'completed', completed_at = ?
		WHERE id = ? AND user_id = ? AND status = 'pending'
	`, sizeBytes, contentType, width, height, now.UTC().Format(time.RFC3339Nano), id, userID)
	if err != nil {
		return domain.ImageUploadRecord{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return domain.ImageUploadRecord{}, err
	}
	if affected == 0 {
		record, getErr := r.GetImageUpload(ctx, userID, id)
		if getErr != nil {
			return domain.ImageUploadRecord{}, getErr
		}
		if record.Status == "completed" && record.SizeBytes == sizeBytes && record.ContentType == contentType && record.Width == width && record.Height == height {
			return record, nil
		}
		return domain.ImageUploadRecord{}, sql.ErrNoRows
	}
	return r.GetImageUpload(ctx, userID, id)
}

func (r *ForumRepository) CreateAuthSession(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	now := nowString()
	_, err := r.exec(ctx, `
		INSERT INTO auth_sessions (user_id, token_hash, created_at, expires_at)
		VALUES (?, ?, ?, ?)
	`, userID, tokenHash, now, expiresAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (r *ForumRepository) GetUserBySessionTokenHash(ctx context.Context, tokenHash string, now time.Time) (domain.User, error) {
	return scanUser(r.queryRow(ctx, `
		SELECT u.id, u.email, u.nickname, u.role, u.province, u.grade, u.created_at
		FROM auth_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ?
		  AND s.revoked_at IS NULL
		  AND s.expires_at > ?
		  AND u.deleted_at IS NULL
		  AND u.banned_at IS NULL
	`, tokenHash, now.UTC().Format(time.RFC3339Nano)))
}

func (r *ForumRepository) RevokeAuthSession(ctx context.Context, tokenHash string, now time.Time) error {
	_, err := r.exec(ctx, `
		UPDATE auth_sessions
		SET revoked_at = ?
		WHERE token_hash = ? AND revoked_at IS NULL
	`, now.UTC().Format(time.RFC3339Nano), tokenHash)
	return err
}

func (r *ForumRepository) RevokeAuthSessionsForUser(ctx context.Context, userID int64, now time.Time) error {
	_, err := r.exec(ctx, `
		UPDATE auth_sessions
		SET revoked_at = ?
		WHERE user_id = ? AND revoked_at IS NULL
	`, now.UTC().Format(time.RFC3339Nano), userID)
	return err
}

func (r *ForumRepository) ListAuthSessions(ctx context.Context, userID int64, currentTokenHash string, now time.Time) ([]domain.AccountSession, error) {
	rows, err := r.query(ctx, `
		SELECT id, token_hash, created_at, expires_at, revoked_at
		FROM auth_sessions
		WHERE user_id = ?
		ORDER BY COALESCE(revoked_at, expires_at) DESC, id DESC
		LIMIT 50
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.AccountSession, 0)
	for rows.Next() {
		var item domain.AccountSession
		var tokenHash string
		var createdAt, expiresAt string
		var revokedAt sql.NullString
		if err := rows.Scan(&item.ID, &tokenHash, &createdAt, &expiresAt, &revokedAt); err != nil {
			return nil, err
		}
		item.CreatedAt = parseTime(createdAt)
		item.ExpiresAt = parseTime(expiresAt)
		if revokedAt.Valid {
			value := parseTime(revokedAt.String)
			item.RevokedAt = &value
		}
		item.Current = tokenHash == currentTokenHash && item.RevokedAt == nil && item.ExpiresAt.After(now)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *ForumRepository) RevokeAuthSessionByID(ctx context.Context, userID int64, sessionID int64, now time.Time) error {
	result, err := r.exec(ctx, `
		UPDATE auth_sessions
		SET revoked_at = ?
		WHERE id = ? AND user_id = ? AND revoked_at IS NULL
	`, now.UTC().Format(time.RFC3339Nano), sessionID, userID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *ForumRepository) CreateEmailVerificationCode(ctx context.Context, email string, codeHash string, expiresAt time.Time) error {
	now := nowString()
	if _, err := r.exec(ctx, `
		UPDATE email_verification_codes
		SET used_at = ?
		WHERE lower(email) = lower(?) AND used_at IS NULL
	`, now, email); err != nil {
		return err
	}
	_, err := r.exec(ctx, `
		INSERT INTO email_verification_codes (email, code_hash, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`, email, codeHash, expiresAt.UTC().Format(time.RFC3339Nano), now)
	return err
}

func (r *ForumRepository) ReserveEmailVerificationAttempt(
	ctx context.Context,
	email string,
	clientIP string,
	now time.Time,
	cooldown time.Duration,
	emailHourlyLimit int,
	ipHourlyLimit int,
) (domain.EmailVerificationAttemptLimit, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	clientIP = strings.TrimSpace(clientIP)
	nowValue := now.UTC()
	cooldownCutoff := now.Add(-cooldown).UTC()
	hourCutoff := now.Add(-time.Hour).UTC()

	if _, err := r.exec(ctx, `DELETE FROM email_verification_attempts WHERE created_at < ?`,
		now.Add(-24*time.Hour).UTC(),
	); err != nil {
		return domain.EmailVerificationAttemptLimit{}, err
	}

	result, err := r.exec(ctx, `
		INSERT INTO email_verification_attempts (email, client_ip, created_at)
		SELECT ?, ?, ?
		WHERE NOT EXISTS (
			SELECT 1
			FROM email_verification_attempts
			WHERE email = ? AND created_at > ?
		)
		  AND (
			SELECT COUNT(*)
			FROM email_verification_attempts
			WHERE email = ? AND created_at > ?
		  ) < ?
		  AND (
			? = '' OR (
				SELECT COUNT(*)
				FROM email_verification_attempts
				WHERE client_ip = ? AND created_at > ?
			) < ?
		  )
	`, email, clientIP, nowValue,
		email, cooldownCutoff,
		email, hourCutoff, emailHourlyLimit,
		clientIP, clientIP, hourCutoff, ipHourlyLimit,
	)
	if err != nil {
		return domain.EmailVerificationAttemptLimit{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return domain.EmailVerificationAttemptLimit{}, err
	}

	emailStats, err := r.emailVerificationAttemptStats(ctx, "email", email, hourCutoff)
	if err != nil {
		return domain.EmailVerificationAttemptLimit{}, err
	}
	ipStats := verificationAttemptStats{}
	if clientIP != "" {
		ipStats, err = r.emailVerificationAttemptStats(ctx, "client_ip", clientIP, hourCutoff)
		if err != nil {
			return domain.EmailVerificationAttemptLimit{}, err
		}
	}

	remaining := emailHourlyLimit - emailStats.count
	if remaining < 0 {
		remaining = 0
	}
	limit := domain.EmailVerificationAttemptLimit{
		Allowed:              affected == 1,
		RetryAfterSeconds:    int(cooldown.Seconds()),
		EmailHourlyLimit:     emailHourlyLimit,
		EmailHourlyRemaining: remaining,
	}
	if limit.Allowed {
		return limit, nil
	}

	retryAt := now.Add(time.Second)
	limit.Scope = "cooldown"
	if emailStats.latest.Valid {
		cooldownRetryAt := emailStats.latest.Time.Add(cooldown)
		if cooldownRetryAt.After(retryAt) {
			retryAt = cooldownRetryAt
		}
	}
	if emailStats.count >= emailHourlyLimit && emailStats.earliest.Valid {
		emailRetryAt := emailStats.earliest.Time.Add(time.Hour)
		if emailRetryAt.After(retryAt) {
			retryAt = emailRetryAt
			limit.Scope = "email_hourly"
		}
	}
	if clientIP != "" && ipStats.count >= ipHourlyLimit && ipStats.earliest.Valid {
		ipRetryAt := ipStats.earliest.Time.Add(time.Hour)
		if ipRetryAt.After(retryAt) {
			retryAt = ipRetryAt
			limit.Scope = "ip_hourly"
		}
	}
	retryDuration := retryAt.Sub(now)
	retryAfter := int((retryDuration + time.Second - 1) / time.Second)
	if retryAfter < 1 {
		retryAfter = 1
	}
	limit.RetryAfterSeconds = retryAfter
	return limit, nil
}

type verificationAttemptStats struct {
	count    int
	earliest sql.NullTime
	latest   sql.NullTime
}

func (r *ForumRepository) emailVerificationAttemptStats(
	ctx context.Context,
	column string,
	value string,
	since time.Time,
) (verificationAttemptStats, error) {
	if column != "email" && column != "client_ip" {
		return verificationAttemptStats{}, errors.New("invalid verification attempt column")
	}
	var stats verificationAttemptStats
	err := r.queryRow(ctx, fmt.Sprintf(`
		SELECT COUNT(*), MIN(created_at), MAX(created_at)
		FROM email_verification_attempts
		WHERE %s = ? AND created_at > ?
	`, column), value, since).Scan(&stats.count, &stats.earliest, &stats.latest)
	return stats, err
}

func (r *ForumRepository) ConsumeEmailVerificationCode(ctx context.Context, email string, codeHash string, maxAttempts int) error {
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	now := nowString()
	transaction, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer transaction.Rollback()

	result, err := execTx(ctx, transaction, `
		UPDATE email_verification_codes
		SET used_at = ?
		WHERE id = (
			SELECT id
			FROM email_verification_codes
			WHERE lower(email) = lower(?)
			  AND code_hash = ?
			  AND used_at IS NULL
			  AND expires_at > ?
			  AND failed_attempts < ?
			ORDER BY created_at DESC
			LIMIT 1
		)
	`, now, email, codeHash, now, maxAttempts)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected > 0 {
		return transaction.Commit()
	}

	if _, err := execTx(ctx, transaction, `
		UPDATE email_verification_codes
		SET failed_attempts = failed_attempts + 1,
		    used_at = CASE WHEN failed_attempts + 1 >= ? THEN ? ELSE used_at END
		WHERE id = (
			SELECT id
			FROM email_verification_codes
			WHERE lower(email) = lower(?)
			  AND used_at IS NULL
			  AND expires_at > ?
			  AND failed_attempts < ?
			ORDER BY created_at DESC
			LIMIT 1
		)
	`, maxAttempts, now, email, now, maxAttempts); err != nil {
		return err
	}
	if err := transaction.Commit(); err != nil {
		return err
	}
	return sql.ErrNoRows
}

func (r *ForumRepository) TogglePostLike(ctx context.Context, userID int64, postID int64) (domain.ToggleResult, error) {
	result, err := r.togglePostRelation(ctx, "post_likes", "likes_count", userID, postID)
	if err == nil && result.Active {
		_ = r.notifyPostOwner(ctx, userID, postID, "like", "赞了你的帖子")
	}
	return result, err
}

func (r *ForumRepository) TogglePostFavorite(ctx context.Context, userID int64, postID int64) (domain.ToggleResult, error) {
	result, err := r.togglePostRelation(ctx, "post_favorites", "favorites_count", userID, postID)
	if err == nil && result.Active {
		_ = r.notifyPostOwner(ctx, userID, postID, "favorite", "收藏了你的帖子")
	}
	return result, err
}

func (r *ForumRepository) ToggleFollowAuthor(ctx context.Context, followerID int64, authorName string) (bool, error) {
	var followerName string
	if err := r.queryRow(ctx, `SELECT nickname FROM users WHERE id = ? AND deleted_at IS NULL`, followerID).Scan(&followerName); err != nil {
		return false, err
	}
	if followerName == authorName {
		return false, errors.New("cannot follow yourself")
	}
	var exists int
	if err := r.queryRow(ctx, `
		SELECT 1
		FROM posts p
		WHERE p.author_name = ? AND p.deleted_at IS NULL
		  AND NOT EXISTS (SELECT 1 FROM content_sources cs WHERE cs.post_id = p.id)
		LIMIT 1
	`, authorName).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, sql.ErrNoRows
		}
		return false, err
	}

	result, err := r.exec(ctx, `DELETE FROM follows WHERE follower_id = ? AND author_name = ?`, followerID, authorName)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if affected > 0 {
		return false, nil
	}
	_, err = r.exec(ctx, `
		INSERT INTO follows (follower_id, author_name, created_at)
		VALUES (?, ?, ?)
	`, followerID, authorName, nowString())
	if err != nil {
		return false, err
	}
	var recipientID int64
	if err := r.queryRow(ctx, `SELECT id FROM users WHERE nickname = ? AND deleted_at IS NULL LIMIT 1`, authorName).Scan(&recipientID); err == nil && recipientID != followerID {
		var actorName string
		_ = r.queryRow(ctx, `SELECT nickname FROM users WHERE id = ?`, followerID).Scan(&actorName)
		_ = r.createNotification(ctx, recipientID, &followerID, "follow", actorName+" 关注了你", "你的公开讨论获得了一位新关注者。", fmt.Sprintf("/users/%s", authorName))
	}
	return true, nil
}

func (r *ForumRepository) GetAccountProfile(ctx context.Context, viewerID *int64, name string) (domain.AccountProfile, error) {
	user, err := scanUser(r.queryRow(ctx, `
		SELECT id, email, nickname, role, province, grade, created_at
		FROM users
		WHERE nickname = ? AND deleted_at IS NULL
		ORDER BY EXISTS (
			SELECT 1 FROM posts p WHERE p.user_id = users.id AND p.deleted_at IS NULL
		) DESC, is_shadow ASC, id ASC
		LIMIT 1
	`, name))
	if err != nil {
		return domain.AccountProfile{}, err
	}
	return r.getAccountProfile(ctx, viewerID, user)
}

func (r *ForumRepository) GetAccountProfileByUserID(ctx context.Context, viewerID *int64, userID int64) (domain.AccountProfile, error) {
	user, err := r.GetUserByID(ctx, userID)
	if err != nil {
		return domain.AccountProfile{}, err
	}
	return r.getAccountProfile(ctx, viewerID, user)
}

func (r *ForumRepository) getAccountProfile(ctx context.Context, viewerID *int64, user domain.User) (domain.AccountProfile, error) {
	profile := domain.AccountProfile{User: user, ChoiceProfile: defaultChoiceProfile()}
	var err error
	var profileJSON string
	if err := r.queryRow(ctx, `SELECT bio, choice_profile FROM user_profiles WHERE user_id = ?`, user.ID).Scan(&profile.Bio, &profileJSON); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return domain.AccountProfile{}, err
	}
	isOwner := viewerID != nil && *viewerID == user.ID
	if isOwner && profileJSON != "" {
		_ = json.Unmarshal([]byte(profileJSON), &profile.ChoiceProfile)
	}

	countQueries := []struct {
		query string
		dest  *int
		arg   any
	}{
		{`SELECT COUNT(*) FROM posts WHERE user_id = ? AND deleted_at IS NULL`, &profile.Stats.Posts, user.ID},
		{`SELECT COUNT(*) FROM comments WHERE user_id = ? AND deleted_at IS NULL`, &profile.Stats.Comments, user.ID},
		{`SELECT COUNT(*) FROM follows WHERE follower_id = ?`, &profile.Stats.Following, user.ID},
		{`SELECT COUNT(*) FROM follows WHERE author_name = ?`, &profile.Stats.Followers, user.Nickname},
		{`SELECT COUNT(*) FROM post_favorites WHERE user_id = ?`, &profile.Stats.Favorites, user.ID},
		{`SELECT COALESCE(SUM(likes_count + favorites_count), 0) FROM posts WHERE user_id = ? AND deleted_at IS NULL`, &profile.Stats.Engagement, user.ID},
	}
	for _, item := range countQueries {
		if err := r.queryRow(ctx, item.query, item.arg).Scan(item.dest); err != nil {
			return domain.AccountProfile{}, err
		}
	}
	postPage, err := r.ListPosts(ctx, viewerID, domain.FeedFilter{UserID: &user.ID, Sort: domain.SortLatest, Limit: 50})
	if err != nil {
		return domain.AccountProfile{}, err
	}
	profile.Posts = postPage.Items
	profile.Comments, err = r.listProfileComments(ctx, user.ID)
	if err != nil {
		return domain.AccountProfile{}, err
	}
	if viewerID != nil {
		var followed int
		err = r.queryRow(ctx, `SELECT 1 FROM follows WHERE follower_id = ? AND author_name = ?`, *viewerID, user.Nickname).Scan(&followed)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return domain.AccountProfile{}, err
		}
		profile.ViewerFollowing = err == nil
	}
	if isOwner {
		profile.Favorites, err = r.listFavoritePosts(ctx, user.ID)
		if err != nil {
			return domain.AccountProfile{}, err
		}
		profile.Following, err = r.listFollowing(ctx, user.ID)
		if err != nil {
			return domain.AccountProfile{}, err
		}
		profile.Followers, err = r.listFollowers(ctx, user.Nickname)
		if err != nil {
			return domain.AccountProfile{}, err
		}
	} else {
		profile.User.Email = ""
	}
	return profile, nil
}

func (r *ForumRepository) listFollowing(ctx context.Context, userID int64) ([]domain.FollowProfile, error) {
	rows, err := r.query(ctx, `
		SELECT f.author_name,
		       COALESCE(u.role, p.author_role, 'student'),
		       COALESCE(u.province, p.province, '未公开'),
		       COALESCE(u.grade, p.grade, '选科用户'),
		       f.created_at
		FROM follows f
		LEFT JOIN users u ON u.nickname = f.author_name AND u.deleted_at IS NULL
		LEFT JOIN posts p ON p.id = (
			SELECT id FROM posts
			WHERE author_name = f.author_name AND deleted_at IS NULL
			ORDER BY created_at DESC LIMIT 1
		)
		WHERE f.follower_id = ?
		ORDER BY f.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFollowProfiles(rows)
}

func (r *ForumRepository) listFollowers(ctx context.Context, authorName string) ([]domain.FollowProfile, error) {
	rows, err := r.query(ctx, `
		SELECT u.nickname, u.role, u.province, u.grade, f.created_at
		FROM follows f
		JOIN users u ON u.id = f.follower_id AND u.deleted_at IS NULL
		WHERE f.author_name = ?
		ORDER BY f.created_at DESC
	`, authorName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFollowProfiles(rows)
}

func scanFollowProfiles(rows *sql.Rows) ([]domain.FollowProfile, error) {
	items := make([]domain.FollowProfile, 0)
	for rows.Next() {
		var item domain.FollowProfile
		var followedAt string
		if err := rows.Scan(&item.Name, &item.Role, &item.Province, &item.Grade, &followedAt); err != nil {
			return nil, err
		}
		item.FollowedAt = parseTime(followedAt)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *ForumRepository) ListConversations(ctx context.Context, userID int64, limit int, cursor string) (domain.ConversationPage, error) {
	cursorTime, cursorID, err := parseConversationCursor(cursor)
	if err != nil {
		return domain.ConversationPage{}, err
	}
	query := `
		WITH ranked_messages AS (
			SELECT m.*,
			       CASE WHEN m.sender_user_id = ? THEN m.recipient_user_id ELSE m.sender_user_id END AS peer_id,
			       ROW_NUMBER() OVER (
				PARTITION BY CASE WHEN m.sender_user_id = ? THEN m.recipient_user_id ELSE m.sender_user_id END
				ORDER BY m.created_at DESC, m.id DESC
			       ) AS row_number
			FROM direct_messages m
			WHERE m.sender_user_id = ? OR m.recipient_user_id = ?
		)
		SELECT u.id, u.nickname, u.role, u.province, u.grade,
		       latest.content, latest.created_at, latest.id,
		       (SELECT COUNT(*)
		          FROM direct_messages unread
		         WHERE unread.sender_user_id = u.id
		           AND unread.recipient_user_id = ?
		           AND unread.read_at IS NULL) AS unread_count
		FROM users u
		JOIN ranked_messages latest ON latest.peer_id = u.id AND latest.row_number = 1
		WHERE u.deleted_at IS NULL`
	args := []any{userID, userID, userID, userID, userID}
	if cursor != "" {
		query += ` AND (latest.created_at < ? OR (latest.created_at = ? AND latest.id < ?))`
		args = append(args, cursorTime, cursorTime, cursorID)
	}
	query += ` ORDER BY latest.created_at DESC, latest.id DESC LIMIT ?`
	args = append(args, limit+1)
	rows, err := r.query(ctx, query, args...)
	if err != nil {
		return domain.ConversationPage{}, err
	}
	defer rows.Close()
	items := make([]domain.Conversation, 0)
	messageIDs := make([]int64, 0)
	for rows.Next() {
		var user domain.User
		var content, createdAt string
		var messageID int64
		var unreadCount int
		if err := rows.Scan(&user.ID, &user.Nickname, &user.Role, &user.Province, &user.Grade, &content, &createdAt, &messageID, &unreadCount); err != nil {
			return domain.ConversationPage{}, err
		}
		user.PublicID = formatUserPublicID(user.ID)
		user.Email = ""
		user.CreatedAt = time.Time{}
		items = append(items, domain.Conversation{User: user, LastMessage: content, LastMessageAt: parseTime(createdAt), UnreadCount: unreadCount})
		messageIDs = append(messageIDs, messageID)
	}
	if err := rows.Err(); err != nil {
		return domain.ConversationPage{}, err
	}
	page := domain.ConversationPage{Items: items}
	if len(items) > limit {
		page.HasMore = true
		page.Items = items[:limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = conversationCursor(last.LastMessageAt, messageIDs[limit-1])
	}
	return page, nil
}

func (r *ForumRepository) ListDirectMessages(ctx context.Context, userID int64, peerName string, limit int, cursor string) (domain.DirectMessagePage, error) {
	peer, err := r.getUserByNickname(ctx, peerName)
	if err != nil {
		return domain.DirectMessagePage{}, err
	}
	if _, err := r.exec(ctx, `
		UPDATE direct_messages SET read_at = COALESCE(read_at, ?)
		WHERE sender_user_id = ? AND recipient_user_id = ?
	`, nowString(), peer.ID, userID); err != nil {
		return domain.DirectMessagePage{}, err
	}
	cursorTime, cursorID, err := parseDirectMessageCursor(cursor)
	if err != nil {
		return domain.DirectMessagePage{}, err
	}
	query := `
		SELECT m.id, m.sender_user_id, sender.nickname, m.recipient_user_id, recipient.nickname,
		       m.content, m.created_at, m.read_at
		FROM direct_messages m
		JOIN users sender ON sender.id = m.sender_user_id
		JOIN users recipient ON recipient.id = m.recipient_user_id
		WHERE ((m.sender_user_id = ? AND m.recipient_user_id = ?)
		   OR (m.sender_user_id = ? AND m.recipient_user_id = ?))`
	args := []any{userID, peer.ID, peer.ID, userID}
	if cursor != "" {
		query += ` AND (m.created_at < ? OR (m.created_at = ? AND m.id < ?))`
		args = append(args, cursorTime, cursorTime, cursorID)
	}
	query += ` ORDER BY m.created_at DESC, m.id DESC LIMIT ?`
	args = append(args, limit+1)
	rows, err := r.query(ctx, query, args...)
	if err != nil {
		return domain.DirectMessagePage{}, err
	}
	defer rows.Close()
	items := make([]domain.DirectMessage, 0)
	for rows.Next() {
		item, err := scanDirectMessage(rows)
		if err != nil {
			return domain.DirectMessagePage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.DirectMessagePage{}, err
	}
	page := domain.DirectMessagePage{Items: items}
	if len(items) > limit {
		page.HasMore = true
		page.Items = items[:limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = directMessageCursor(last.CreatedAt, last.ID)
	}
	sort.Slice(page.Items, func(i, j int) bool {
		left := page.Items[i]
		right := page.Items[j]
		if left.CreatedAt.Equal(right.CreatedAt) {
			return left.ID < right.ID
		}
		return left.CreatedAt.Before(right.CreatedAt)
	})
	return page, nil
}

func (r *ForumRepository) SendDirectMessage(ctx context.Context, senderID int64, recipientName string, content string) (domain.DirectMessage, error) {
	recipient, err := r.getUserByNickname(ctx, recipientName)
	if err != nil {
		return domain.DirectMessage{}, err
	}
	if recipient.ID == senderID {
		return domain.DirectMessage{}, errors.New("cannot message yourself")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.DirectMessage{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var senderName string
	if err := queryRowTx(ctx, tx, `SELECT nickname FROM users WHERE id = ? AND deleted_at IS NULL`, senderID).Scan(&senderName); err != nil {
		return domain.DirectMessage{}, err
	}
	now := nowString()
	var id int64
	err = queryRowTx(ctx, tx, `
		INSERT INTO direct_messages (sender_user_id, recipient_user_id, content, created_at)
		VALUES (?, ?, ?, ?)
		RETURNING id
	`, senderID, recipient.ID, content, now).Scan(&id)
	if err != nil {
		return domain.DirectMessage{}, err
	}
	if _, err := execTx(ctx, tx, `
		INSERT INTO notifications (recipient_user_id, actor_user_id, type, title, summary, target_url, created_at)
		VALUES (?, ?, 'message', ?, ?, '/messages', ?)
	`, recipient.ID, senderID, senderName+" 给你发来私信", content, now); err != nil {
		return domain.DirectMessage{}, err
	}
	message, err := scanDirectMessage(queryRowTx(ctx, tx, `
		SELECT m.id, m.sender_user_id, sender.nickname, m.recipient_user_id, recipient.nickname,
		       m.content, m.created_at, m.read_at
		FROM direct_messages m
		JOIN users sender ON sender.id = m.sender_user_id
		JOIN users recipient ON recipient.id = m.recipient_user_id
		WHERE m.id = ?
	`, id))
	if err != nil {
		return domain.DirectMessage{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.DirectMessage{}, err
	}
	return message, nil
}

func (r *ForumRepository) getUserByNickname(ctx context.Context, name string) (domain.User, error) {
	return scanUser(r.queryRow(ctx, `
		SELECT id, email, nickname, role, province, grade, created_at
		FROM users WHERE nickname = ? AND deleted_at IS NULL LIMIT 1
	`, name))
}

func (r *ForumRepository) getDirectMessage(ctx context.Context, id int64) (domain.DirectMessage, error) {
	return scanDirectMessage(r.queryRow(ctx, `
		SELECT m.id, m.sender_user_id, sender.nickname, m.recipient_user_id, recipient.nickname,
		       m.content, m.created_at, m.read_at
		FROM direct_messages m
		JOIN users sender ON sender.id = m.sender_user_id
		JOIN users recipient ON recipient.id = m.recipient_user_id
		WHERE m.id = ?
	`, id))
}

type directMessageScanner interface {
	Scan(dest ...any) error
}

func scanDirectMessage(scanner directMessageScanner) (domain.DirectMessage, error) {
	var item domain.DirectMessage
	var createdAt string
	var readAt sql.NullString
	if err := scanner.Scan(&item.ID, &item.SenderID, &item.SenderName, &item.RecipientID, &item.RecipientName, &item.Content, &createdAt, &readAt); err != nil {
		return domain.DirectMessage{}, err
	}
	item.CreatedAt = parseTime(createdAt)
	if readAt.Valid {
		value := parseTime(readAt.String)
		item.ReadAt = &value
	}
	return item, nil
}

func (r *ForumRepository) UpdateAccountProfile(ctx context.Context, userID int64, input domain.UpdateProfileInput) (domain.AccountProfile, error) {
	now := nowString()
	_, err := r.exec(ctx, `
		INSERT INTO user_profiles (user_id, bio, choice_profile, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET bio = excluded.bio, choice_profile = excluded.choice_profile, updated_at = excluded.updated_at
	`, userID, strings.TrimSpace(input.Bio), mustJSON(input.ChoiceProfile), now, now)
	if err != nil {
		return domain.AccountProfile{}, err
	}
	user, err := r.GetUserByID(ctx, userID)
	if err != nil {
		return domain.AccountProfile{}, err
	}
	return r.GetAccountProfileByUserID(ctx, &userID, user.ID)
}

func (r *ForumRepository) ListNotifications(ctx context.Context, userID int64, limit int, cursor string) (domain.NotificationPage, error) {
	cursorTime, cursorID, err := parseNotificationCursor(cursor)
	if err != nil {
		return domain.NotificationPage{}, err
	}
	query := `
		SELECT n.id, n.type, n.title, n.summary, n.target_url, COALESCE(u.nickname, ''), n.created_at, n.read_at
		FROM notifications n
		LEFT JOIN users u ON u.id = n.actor_user_id
		WHERE n.recipient_user_id = ?`
	args := []any{userID}
	if cursor != "" {
		query += ` AND (n.created_at < ? OR (n.created_at = ? AND n.id < ?))`
		args = append(args, cursorTime, cursorTime, cursorID)
	}
	query += ` ORDER BY n.created_at DESC, n.id DESC LIMIT ?`
	args = append(args, limit+1)
	rows, err := r.query(ctx, query, args...)
	if err != nil {
		return domain.NotificationPage{}, err
	}
	defer rows.Close()
	items := make([]domain.Notification, 0)
	for rows.Next() {
		var item domain.Notification
		var createdAt string
		var readAt sql.NullString
		if err := rows.Scan(&item.ID, &item.Type, &item.Title, &item.Summary, &item.TargetURL, &item.ActorName, &createdAt, &readAt); err != nil {
			return domain.NotificationPage{}, err
		}
		item.CreatedAt = parseTime(createdAt)
		if readAt.Valid {
			value := parseTime(readAt.String)
			item.ReadAt = &value
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.NotificationPage{}, err
	}
	page := domain.NotificationPage{Items: items}
	if len(items) > limit {
		page.HasMore = true
		page.Items = items[:limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = notificationCursor(last.CreatedAt, last.ID)
	}
	return page, nil
}

func notificationCursor(createdAt time.Time, id int64) string {
	return createdAt.UTC().Format(time.RFC3339Nano) + "_" + fmt.Sprintf("%d", id)
}

func parseNotificationCursor(cursor string) (string, int64, error) {
	if cursor == "" {
		return "", 0, nil
	}
	index := strings.LastIndex(cursor, "_")
	if index <= 0 || index == len(cursor)-1 {
		return "", 0, fmt.Errorf("invalid notification cursor")
	}
	id, err := strconv.ParseInt(cursor[index+1:], 10, 64)
	if err != nil || id <= 0 {
		return "", 0, fmt.Errorf("invalid notification cursor")
	}
	return cursor[:index], id, nil
}

func postCursor(createdAt time.Time, id int64) string {
	return createdAt.UTC().Format(time.RFC3339Nano) + "_" + fmt.Sprintf("%d", id)
}

func parsePostCursor(cursor string) (string, int64, error) {
	return parseNotificationCursor(cursor)
}

func directMessageCursor(createdAt time.Time, id int64) string {
	return postCursor(createdAt, id)
}

func parseDirectMessageCursor(cursor string) (string, int64, error) {
	return parseNotificationCursor(cursor)
}

func conversationCursor(createdAt time.Time, id int64) string {
	return postCursor(createdAt, id)
}

func parseConversationCursor(cursor string) (string, int64, error) {
	createdAt, id, err := parseNotificationCursor(cursor)
	if err != nil {
		return "", 0, fmt.Errorf("invalid conversation cursor")
	}
	return createdAt, id, nil
}

func (r *ForumRepository) MarkNotificationRead(ctx context.Context, userID int64, notificationID *int64) error {
	query := `UPDATE notifications SET read_at = COALESCE(read_at, ?) WHERE recipient_user_id = ?`
	args := []any{nowString(), userID}
	if notificationID != nil {
		query += ` AND id = ?`
		args = append(args, *notificationID)
	}
	_, err := r.exec(ctx, query, args...)
	return err
}

func (r *ForumRepository) togglePostRelation(ctx context.Context, table string, counter string, userID int64, postID int64) (domain.ToggleResult, error) {
	if _, err := r.fetchPostByID(ctx, postID); err != nil {
		return domain.ToggleResult{}, err
	}
	result, err := r.exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE user_id = ? AND post_id = ?`, table), userID, postID)
	if err != nil {
		return domain.ToggleResult{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return domain.ToggleResult{}, err
	}

	active := false
	delta := -1
	if affected == 0 {
		if _, err := r.exec(ctx, fmt.Sprintf(`INSERT INTO %s (user_id, post_id, created_at) VALUES (?, ?, ?)`, table),
			userID,
			postID,
			nowString(),
		); err != nil {
			return domain.ToggleResult{}, err
		}
		active = true
		delta = 1
	}

	if _, err := r.exec(ctx, fmt.Sprintf(`UPDATE posts SET %s = GREATEST(%s + ?, 0), updated_at = ? WHERE id = ?`, counter, counter),
		delta,
		nowString(),
		postID,
	); err != nil {
		return domain.ToggleResult{}, err
	}

	var count int
	if err := r.queryRow(ctx, fmt.Sprintf(`SELECT %s FROM posts WHERE id = ?`, counter), postID).Scan(&count); err != nil {
		return domain.ToggleResult{}, err
	}
	return domain.ToggleResult{Active: active, Count: count}, nil
}

func (r *ForumRepository) createNotification(ctx context.Context, recipientID int64, actorID *int64, notificationType string, title string, summary string, targetURL string) error {
	_, err := r.exec(ctx, `
		INSERT INTO notifications (recipient_user_id, actor_user_id, type, title, summary, target_url, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, recipientID, actorID, notificationType, title, summary, targetURL, nowString())
	return err
}

func (r *ForumRepository) notifyPostOwner(ctx context.Context, actorID int64, postID int64, notificationType string, action string) error {
	post, err := r.fetchPostByID(ctx, postID)
	if err != nil || post.UserID == nil || *post.UserID == actorID {
		return err
	}
	var actorName string
	if err := r.queryRow(ctx, `SELECT nickname FROM users WHERE id = ?`, actorID).Scan(&actorName); err != nil {
		return err
	}
	return r.createNotification(ctx, *post.UserID, &actorID, notificationType, actorName+" "+action, post.Title, fmt.Sprintf("/posts/%d", postID))
}

func (r *ForumRepository) listProfileComments(ctx context.Context, userID int64) ([]domain.ProfileComment, error) {
	rows, err := r.query(ctx, `
		SELECT c.id, c.post_id, c.user_id, c.author, c.role, c.content, c.created_at, p.title
		FROM comments c
		JOIN posts p ON p.id = c.post_id
		WHERE c.user_id = ? AND c.deleted_at IS NULL AND p.deleted_at IS NULL
		ORDER BY c.created_at DESC
		LIMIT 50
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.ProfileComment, 0)
	for rows.Next() {
		var item domain.ProfileComment
		var commentUserID sql.NullInt64
		var createdAt string
		if err := rows.Scan(&item.Comment.ID, &item.Comment.PostID, &commentUserID, &item.Comment.Author, &item.Comment.Role, &item.Comment.Content, &createdAt, &item.PostTitle); err != nil {
			return nil, err
		}
		if commentUserID.Valid {
			value := commentUserID.Int64
			item.Comment.UserID = &value
		}
		item.Comment.CreatedAt = parseTime(createdAt)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *ForumRepository) listFavoritePosts(ctx context.Context, userID int64) ([]domain.Post, error) {
	rows, err := r.query(ctx, `
		SELECT p.id, p.user_id, p.author_name, p.author_role, p.title, p.content, p.image_urls, p.tags, p.track, p.electives,
		       p.category, p.grade, p.province, p.likes_count, p.comments_count, p.favorites_count, p.created_at, p.updated_at,
		       COALESCE(cs.source_platform, ''), COALESCE(cs.source_url, ''), COALESCE(cs.source_title, ''),
		       COALESCE(cs.source_author, ''), COALESCE(cs.source_avatar_url, '')
		FROM post_favorites pf
		JOIN posts p ON p.id = pf.post_id
		LEFT JOIN content_sources cs ON cs.post_id = p.id
		WHERE pf.user_id = ? AND p.deleted_at IS NULL
		ORDER BY pf.created_at DESC
		LIMIT 50
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.Post, 0)
	for rows.Next() {
		post, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		post.ViewerFavorited = true
		items = append(items, post)
	}
	return items, rows.Err()
}

func defaultChoiceProfile() domain.ChoiceProfile {
	return domain.ChoiceProfile{
		SchoolType:          "普通高中",
		SubjectStability:    "中等",
		PreferredTrack:      domain.TrackPhysics,
		PreferredSubjects:   []domain.Subject{domain.SubjectChemistry, domain.SubjectBiology},
		LearningStyle:       "理解推导型",
		PressureTolerance:   "中等",
		RecommendationFocus: "专业覆盖率优先",
	}
}

func truncateText(value string, limit int) string {
	text := []rune(strings.TrimSpace(value))
	if len(text) <= limit {
		return string(text)
	}
	return string(text[:limit]) + "…"
}

func (r *ForumRepository) fetchPostByID(ctx context.Context, id int64) (domain.Post, error) {
	return scanPost(r.queryRow(ctx, `
		SELECT p.id, p.user_id, p.author_name, p.author_role, p.title, p.content, p.image_urls, p.tags, p.track, p.electives,
		       p.category, p.grade, p.province, p.likes_count, p.comments_count, p.favorites_count, p.created_at, p.updated_at,
		       COALESCE(cs.source_platform, ''), COALESCE(cs.source_url, ''), COALESCE(cs.source_title, ''),
		       COALESCE(cs.source_author, ''), COALESCE(cs.source_avatar_url, '')
		FROM posts p
		LEFT JOIN content_sources cs ON cs.post_id = p.id
		WHERE p.id = ? AND p.deleted_at IS NULL
	`, id))
}

func (r *ForumRepository) viewerState(ctx context.Context, viewerID *int64) (map[int64]bool, map[int64]bool, map[string]bool, error) {
	liked := map[int64]bool{}
	favorited := map[int64]bool{}
	followed := map[string]bool{}
	if viewerID == nil {
		return liked, favorited, followed, nil
	}

	loadIDs := func(query string, target map[int64]bool) error {
		rows, err := r.query(ctx, query, *viewerID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				return err
			}
			target[id] = true
		}
		return rows.Err()
	}
	if err := loadIDs(`SELECT post_id FROM post_likes WHERE user_id = ?`, liked); err != nil {
		return nil, nil, nil, err
	}
	if err := loadIDs(`SELECT post_id FROM post_favorites WHERE user_id = ?`, favorited); err != nil {
		return nil, nil, nil, err
	}

	rows, err := r.query(ctx, `SELECT author_name FROM follows WHERE follower_id = ?`, *viewerID)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, nil, nil, err
		}
		followed[name] = true
	}
	return liked, favorited, followed, rows.Err()
}

type postScanner interface {
	Scan(dest ...any) error
}

type commentScanner interface {
	Scan(dest ...any) error
}

type insightScanner interface {
	Scan(dest ...any) error
}

type topicScanner interface {
	Scan(dest ...any) error
}

type userScanner interface {
	Scan(dest ...any) error
}

func (r *ForumRepository) scanContentReportByID(ctx context.Context, id int64) (domain.ContentReport, error) {
	return scanContentReport(r.queryRow(ctx, `
		SELECT cr.id, cr.reporter_user_id, COALESCE(u.nickname, ''), cr.target_type, cr.target_id,
		       COALESCE(p.title, ''), COALESCE(p.author_name, ''), cr.reason, cr.detail, cr.status,
		       cr.resolution_note, cr.resolved_at, cr.created_at, cr.updated_at
		FROM content_reports cr
		LEFT JOIN users u ON u.id = cr.reporter_user_id
		LEFT JOIN posts p ON cr.target_type = 'post' AND p.id = cr.target_id
		WHERE cr.id = ?
	`, id))
}

func scanContentReport(scanner interface {
	Scan(dest ...any) error
}) (domain.ContentReport, error) {
	var report domain.ContentReport
	var resolvedAt sql.NullString
	var createdAt, updatedAt string
	err := scanner.Scan(
		&report.ID,
		&report.ReporterID,
		&report.ReporterName,
		&report.TargetType,
		&report.TargetID,
		&report.TargetTitle,
		&report.TargetAuthor,
		&report.Reason,
		&report.Detail,
		&report.Status,
		&report.ResolutionNote,
		&resolvedAt,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return domain.ContentReport{}, err
	}
	if resolvedAt.Valid {
		parsed := parseTime(resolvedAt.String)
		report.ResolvedAt = &parsed
	}
	report.CreatedAt = parseTime(createdAt)
	report.UpdatedAt = parseTime(updatedAt)
	return report, nil
}

func scanPost(scanner postScanner) (domain.Post, error) {
	var post domain.Post
	var userID sql.NullInt64
	var imageURLsRaw string
	var tagsRaw string
	var electivesRaw string
	var createdAt string
	var updatedAt string
	err := scanner.Scan(
		&post.ID,
		&userID,
		&post.AuthorName,
		&post.AuthorRole,
		&post.Title,
		&post.Content,
		&imageURLsRaw,
		&tagsRaw,
		&post.Track,
		&electivesRaw,
		&post.Category,
		&post.Grade,
		&post.Province,
		&post.LikesCount,
		&post.CommentsCount,
		&post.FavoritesCount,
		&createdAt,
		&updatedAt,
		&post.SourcePlatform,
		&post.SourceURL,
		&post.SourceTitle,
		&post.SourceAuthor,
		&post.SourceAvatarURL,
	)
	if err != nil {
		return domain.Post{}, err
	}
	if userID.Valid {
		post.UserID = &userID.Int64
	}
	post.ImageURLs = parseStringSlice(imageURLsRaw)
	post.Tags = parseStringSlice(tagsRaw)
	post.Electives = subjectsFromStrings(parseStringSlice(electivesRaw))
	post.CreatedAt = parseTime(createdAt)
	post.UpdatedAt = parseTime(updatedAt)
	return post, nil
}

func scanComment(scanner commentScanner) (domain.Comment, error) {
	var comment domain.Comment
	var userID sql.NullInt64
	var createdAt string
	err := scanner.Scan(
		&comment.ID,
		&comment.PostID,
		&userID,
		&comment.Author,
		&comment.Role,
		&comment.Content,
		&createdAt,
	)
	if err != nil {
		return domain.Comment{}, err
	}
	if userID.Valid {
		comment.UserID = &userID.Int64
	}
	comment.CreatedAt = parseTime(createdAt)
	return comment, nil
}

func scanInsight(scanner insightScanner) (domain.SubjectInsight, error) {
	var insight domain.SubjectInsight
	var capturedAt, updatedAt string
	err := scanner.Scan(
		&insight.ID,
		&insight.Combination,
		&insight.Trend,
		&insight.Heat,
		&insight.MatchRate,
		&insight.Advice,
		&insight.Details,
		&insight.MetricType,
		&insight.Unit,
		&insight.Province,
		&insight.DataYear,
		&insight.SourceName,
		&insight.SourceURL,
		&insight.Scope,
		&insight.SampleSize,
		&capturedAt,
		&insight.Methodology,
		&updatedAt,
	)
	if err != nil {
		return domain.SubjectInsight{}, err
	}
	insight.CapturedAt = parseTime(capturedAt)
	insight.UpdatedAt = parseTime(updatedAt)
	return insight, nil
}

func scanTopic(scanner topicScanner) (domain.Topic, error) {
	var topic domain.Topic
	var createdAt string
	err := scanner.Scan(
		&topic.ID,
		&topic.Slug,
		&topic.TopicTag,
		&topic.Title,
		&topic.Summary,
		&topic.ViewsCount,
		&topic.PostsCount,
		&createdAt,
	)
	if err != nil {
		return domain.Topic{}, err
	}
	topic.CreatedAt = parseTime(createdAt)
	return topic, nil
}

func scanUser(scanner userScanner) (domain.User, error) {
	var user domain.User
	var email sql.NullString
	var createdAt string
	err := scanner.Scan(
		&user.ID,
		&email,
		&user.Nickname,
		&user.Role,
		&user.Province,
		&user.Grade,
		&createdAt,
	)
	if err != nil {
		return domain.User{}, err
	}
	user.Email = email.String
	user.PublicID = formatUserPublicID(user.ID)
	user.CreatedAt = parseTime(createdAt)
	return user, nil
}

func scanImageUpload(scanner interface {
	Scan(dest ...any) error
}) (domain.ImageUploadRecord, error) {
	var record domain.ImageUploadRecord
	var createdAt, expiresAt string
	var completedAt sql.NullString
	err := scanner.Scan(
		&record.ID,
		&record.UserID,
		&record.AssetKey,
		&record.FileName,
		&record.ContentType,
		&record.Ext,
		&record.SizeBytes,
		&record.Width,
		&record.Height,
		&record.Status,
		&createdAt,
		&expiresAt,
		&completedAt,
	)
	if err != nil {
		return domain.ImageUploadRecord{}, err
	}
	record.CreatedAt = parseTime(createdAt)
	record.ExpiresAt = parseTime(expiresAt)
	if completedAt.Valid {
		parsed := parseTime(completedAt.String)
		record.CompletedAt = &parsed
	}
	return record, nil
}

func formatUserPublicID(id int64) string {
	return fmt.Sprintf("%08d", id)
}

func parseTime(value string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func parseStringSlice(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return []string{}
	}
	return values
}

func sortPosts(posts []domain.Post, sortMode domain.FeedSort) {
	sort.Slice(posts, func(i int, j int) bool {
		left := posts[i]
		right := posts[j]
		switch sortMode {
		case domain.SortLatest:
			return left.CreatedAt.After(right.CreatedAt)
		case domain.SortHot:
			leftHot := left.LikesCount + left.CommentsCount*4
			rightHot := right.LikesCount + right.CommentsCount*4
			if leftHot == rightHot {
				return left.UpdatedAt.After(right.UpdatedAt)
			}
			return leftHot > rightHot
		default:
			leftScore := recommendationScore(left)
			rightScore := recommendationScore(right)
			if leftScore == rightScore {
				return left.CreatedAt.After(right.CreatedAt)
			}
			return leftScore > rightScore
		}
	})
}

func recommendationScore(post domain.Post) float64 {
	score := float64(minInt(post.LikesCount, 300))*0.8 +
		float64(minInt(post.CommentsCount, 80))*4 +
		float64(minInt(post.FavoritesCount, 120))*3
	if post.AuthorRole == "teacher" || post.AuthorRole == "counselor" {
		score += 45
	}
	if post.LikesCount < 150 {
		score += 65
	}
	return score
}

func subjectStrings(subjects []domain.Subject) []string {
	values := make([]string, 0, len(subjects))
	for _, subject := range subjects {
		values = append(values, string(subject))
	}
	return values
}

func subjectsFromStrings(values []string) []domain.Subject {
	subjects := make([]domain.Subject, 0, len(values))
	for _, value := range values {
		subjects = append(subjects, domain.Subject(value))
	}
	return subjects
}

func mustJSON(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func nowString() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func postContentType(category domain.PostCategory) string {
	switch category {
	case domain.CategoryQuestion:
		return "家长提问"
	case domain.CategoryData:
		return "数据建议"
	default:
		return "经验帖"
	}
}
