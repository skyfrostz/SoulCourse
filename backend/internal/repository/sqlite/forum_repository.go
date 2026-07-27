package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
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

func (r *ForumRepository) ListPosts(ctx context.Context, viewerID *int64, filter domain.FeedFilter) ([]domain.Post, error) {
	query, args := buildPostListQuery(filter)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := make([]domain.Post, 0, filter.Limit)
	for rows.Next() {
		post, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	liked, favorited, followed, err := r.viewerState(ctx, viewerID)
	if err != nil {
		return nil, err
	}

	for i := range posts {
		post := &posts[i]
		post.ViewerLiked = liked[post.ID]
		post.ViewerFavorited = favorited[post.ID]
		post.ViewerFollowing = post.SourcePlatform == "" && followed[post.AuthorName]
	}
	return posts, nil
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

	subjects := make([]domain.Subject, 0, len(filter.Subjects)+1)
	subjects = append(subjects, filter.Subjects...)
	if filter.Subject != "" {
		subjects = append(subjects, filter.Subject)
	}
	for _, subject := range subjects {
		if subject == "" {
			continue
		}
		query += ` AND p.electives LIKE ? ESCAPE '\'`
		args = append(args, `%"`+escapeLike(string(subject))+`"%`)
	}

	if keyword := strings.ToLower(strings.TrimSpace(filter.Keyword)); keyword != "" {
		query += ` AND LOWER(p.title || ' ' || p.content || ' ' || p.author_name || ' ' || p.province || ' ' || p.grade || ' ' || p.tags) LIKE ? ESCAPE '\'`
		args = append(args, "%"+escapeLike(keyword)+"%")
	}
	if authorName := strings.TrimSpace(filter.AuthorName); authorName != "" {
		query += " AND p.author_name = ?"
		args = append(args, authorName)
	}

	switch filter.Sort {
	case domain.SortLatest:
		query += " ORDER BY p.created_at DESC, p.id DESC"
	case domain.SortHot:
		query += " ORDER BY p.likes_count + p.comments_count * 4 DESC, p.updated_at DESC, p.id DESC"
	default:
		query += ` ORDER BY
			(CASE WHEN p.title LIKE '%选科%' OR p.tags LIKE '%选科%' THEN 1500 ELSE 0 END
			 + MIN(p.likes_count, 300) * 0.8 + MIN(p.comments_count, 80) * 4 + MIN(p.favorites_count, 120) * 3
			 + CASE WHEN p.author_role IN ('teacher', 'counselor') THEN 45 ELSE 0 END
			 + CASE WHEN p.likes_count < 150 THEN 65 ELSE 0 END) DESC,
			p.created_at DESC, p.id DESC`
	}

	query += " LIMIT ? OFFSET ?"
	args = append(args, filter.Limit, filter.Offset)
	return query, args
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

	rows, err := r.db.QueryContext(ctx, `
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
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO posts (user_id, author_name, author_role, title, content, image_urls, tags, track, electives, category, grade, province, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
	)
	if err != nil {
		return domain.Post{}, err
	}
	postID, err := result.LastInsertId()
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
	_, err = r.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO admin_content_records
			(id, module, title, content_type, status, scope, owner, tags, summary, url, priority, sort_order, payload, created_at, updated_at)
		VALUES (?, 'posts', ?, ?, '已上架', ?, ?, ?, ?, ?, '常规', 0, ?, ?, ?)
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

func (r *ForumRepository) CreateComment(ctx context.Context, user domain.User, postID int64, input domain.CreateCommentInput) (domain.Comment, error) {
	post, err := r.fetchPostByID(ctx, postID)
	if err != nil {
		return domain.Comment{}, err
	}
	now := nowString()
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO comments (post_id, user_id, author, role, content, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, postID, user.ID, user.Nickname, user.Role, input.Content, now)
	if err != nil {
		return domain.Comment{}, err
	}
	commentID, err := result.LastInsertId()
	if err != nil {
		return domain.Comment{}, err
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE posts SET comments_count = comments_count + 1, updated_at = ? WHERE id = ?`, now, postID); err != nil {
		return domain.Comment{}, err
	}
	if post.UserID != nil && *post.UserID != user.ID {
		_ = r.createNotification(ctx, *post.UserID, &user.ID, "comment", user.Nickname+" 评论了你的帖子", truncateText(input.Content, 90), fmt.Sprintf("/posts/%d", postID))
	}
	return scanComment(r.db.QueryRowContext(ctx, `
		SELECT id, post_id, user_id, author, role, content, created_at
		FROM comments
		WHERE id = ?
	`, commentID))
}

func (r *ForumRepository) ListInsights(ctx context.Context) ([]domain.SubjectInsight, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, combination, trend, heat, match_rate, advice, details, updated_at
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
	return scanInsight(r.db.QueryRowContext(ctx, `
		SELECT id, combination, trend, heat, match_rate, advice, details, updated_at
		FROM subject_insights
		WHERE id = ?
	`, id))
}

func (r *ForumRepository) ListTopics(ctx context.Context) ([]domain.Topic, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, slug, title, summary, views_count, posts_count, created_at
		FROM topics
		ORDER BY views_count DESC, id ASC
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
	if _, err := r.db.ExecContext(ctx, `UPDATE topics SET views_count = views_count + 1 WHERE slug = ?`, slug); err != nil {
		return domain.TopicDetail{}, err
	}
	topic, err := scanTopic(r.db.QueryRowContext(ctx, `
		SELECT id, slug, title, summary, views_count, posts_count, created_at
		FROM topics
		WHERE slug = ?
	`, slug))
	if err != nil {
		return domain.TopicDetail{}, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT p.id, p.user_id, p.author_name, p.author_role, p.title, p.content, p.image_urls, p.tags, p.track, p.electives,
		       p.category, p.grade, p.province, p.likes_count, p.comments_count, p.favorites_count, p.created_at, p.updated_at,
		       COALESCE(cs.source_platform, ''), COALESCE(cs.source_url, ''), COALESCE(cs.source_title, ''),
		       COALESCE(cs.source_author, ''), COALESCE(cs.source_avatar_url, '')
		FROM posts p
		JOIN topic_posts tp ON tp.post_id = p.id
		LEFT JOIN content_sources cs ON cs.post_id = p.id
		WHERE tp.topic_id = ? AND p.deleted_at IS NULL
	`, topic.ID)
	if err != nil {
		return domain.TopicDetail{}, err
	}
	defer rows.Close()

	liked, favorited, followed, err := r.viewerState(ctx, viewerID)
	if err != nil {
		return domain.TopicDetail{}, err
	}

	posts := make([]domain.Post, 0)
	for rows.Next() {
		post, err := scanPost(rows)
		if err != nil {
			return domain.TopicDetail{}, err
		}
		post.ViewerLiked = liked[post.ID]
		post.ViewerFavorited = favorited[post.ID]
		post.ViewerFollowing = post.SourcePlatform == "" && followed[post.AuthorName]
		posts = append(posts, post)
	}
	sortPosts(posts, domain.SortLatest)
	return domain.TopicDetail{Topic: topic, Posts: posts}, rows.Err()
}

func (r *ForumRepository) CreateUser(ctx context.Context, input domain.RegisterInput, passwordHash string) (domain.User, error) {
	now := nowString()
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO users (email, password_hash, nickname, role, province, grade, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, input.Email, passwordHash, input.Nickname, input.Role, input.Province, input.Grade, now, now)
	if err != nil {
		return domain.User{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return domain.User{}, err
	}
	nowProfile := nowString()
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO user_profiles (user_id, bio, choice_profile, created_at, updated_at)
		VALUES (?, '', ?, ?, ?)
	`, id, mustJSON(defaultChoiceProfile()), nowProfile, nowProfile); err != nil {
		return domain.User{}, err
	}
	if err := r.createNotification(ctx, id, nil, "profile", "完善你的选科画像", "补充 MBTI、目标专业和学科稳定性，让建议更贴近你。", "/settings"); err != nil {
		return domain.User{}, err
	}
	return scanUser(r.db.QueryRowContext(ctx, `
		SELECT id, email, nickname, role, province, grade, created_at
		FROM users
		WHERE id = ?
	`, id))
}

func (r *ForumRepository) GetUserByEmail(ctx context.Context, email string) (domain.User, string, error) {
	var user domain.User
	var createdAt string
	var passwordHash string
	err := r.db.QueryRowContext(ctx, `
		SELECT id, email, nickname, role, province, grade, created_at, password_hash
		FROM users
		WHERE lower(email) = lower(?) AND deleted_at IS NULL
	`, email).Scan(&user.ID, &user.Email, &user.Nickname, &user.Role, &user.Province, &user.Grade, &createdAt, &passwordHash)
	if err != nil {
		return domain.User{}, "", err
	}
	user.CreatedAt = parseTime(createdAt)
	return user, passwordHash, nil
}

func (r *ForumRepository) GetUserByID(ctx context.Context, id int64) (domain.User, error) {
	return scanUser(r.db.QueryRowContext(ctx, `
		SELECT id, email, nickname, role, province, grade, created_at
		FROM users
		WHERE id = ? AND deleted_at IS NULL
	`, id))
}

func (r *ForumRepository) CreateEmailVerificationCode(ctx context.Context, email string, codeHash string, expiresAt time.Time) error {
	now := nowString()
	if _, err := r.db.ExecContext(ctx, `
		UPDATE email_verification_codes
		SET used_at = ?
		WHERE lower(email) = lower(?) AND used_at IS NULL
	`, now, email); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
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
	nowUnix := now.UTC().Unix()
	cooldownCutoff := now.Add(-cooldown).UTC().Unix()
	hourCutoff := now.Add(-time.Hour).UTC().Unix()

	if _, err := r.db.ExecContext(ctx,
		`DELETE FROM email_verification_attempts WHERE created_at < ?`,
		now.Add(-24*time.Hour).UTC().Unix(),
	); err != nil {
		return domain.EmailVerificationAttemptLimit{}, err
	}

	result, err := r.db.ExecContext(ctx, `
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
	`, email, clientIP, nowUnix,
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
		cooldownRetryAt := time.Unix(emailStats.latest.Int64, 0).Add(cooldown)
		if cooldownRetryAt.After(retryAt) {
			retryAt = cooldownRetryAt
		}
	}
	if emailStats.count >= emailHourlyLimit && emailStats.earliest.Valid {
		emailRetryAt := time.Unix(emailStats.earliest.Int64, 0).Add(time.Hour)
		if emailRetryAt.After(retryAt) {
			retryAt = emailRetryAt
			limit.Scope = "email_hourly"
		}
	}
	if clientIP != "" && ipStats.count >= ipHourlyLimit && ipStats.earliest.Valid {
		ipRetryAt := time.Unix(ipStats.earliest.Int64, 0).Add(time.Hour)
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
	earliest sql.NullInt64
	latest   sql.NullInt64
}

func (r *ForumRepository) emailVerificationAttemptStats(
	ctx context.Context,
	column string,
	value string,
	since int64,
) (verificationAttemptStats, error) {
	if column != "email" && column != "client_ip" {
		return verificationAttemptStats{}, errors.New("invalid verification attempt column")
	}
	var stats verificationAttemptStats
	err := r.db.QueryRowContext(ctx, fmt.Sprintf(`
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

	result, err := transaction.ExecContext(ctx, `
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

	if _, err := transaction.ExecContext(ctx, `
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
	var exists int
	if err := r.db.QueryRowContext(ctx, `
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

	result, err := r.db.ExecContext(ctx, `DELETE FROM follows WHERE follower_id = ? AND author_name = ?`, followerID, authorName)
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
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO follows (follower_id, author_name, created_at)
		VALUES (?, ?, ?)
	`, followerID, authorName, nowString())
	if err != nil {
		return false, err
	}
	var recipientID int64
	if err := r.db.QueryRowContext(ctx, `SELECT id FROM users WHERE nickname = ? AND deleted_at IS NULL LIMIT 1`, authorName).Scan(&recipientID); err == nil && recipientID != followerID {
		var actorName string
		_ = r.db.QueryRowContext(ctx, `SELECT nickname FROM users WHERE id = ?`, followerID).Scan(&actorName)
		_ = r.createNotification(ctx, recipientID, &followerID, "follow", actorName+" 关注了你", "你的公开讨论获得了一位新关注者。", fmt.Sprintf("/users/%s", authorName))
	}
	return true, nil
}

func (r *ForumRepository) GetAccountProfile(ctx context.Context, viewerID *int64, name string) (domain.AccountProfile, error) {
	user, err := scanUser(r.db.QueryRowContext(ctx, `
		SELECT id, email, nickname, role, province, grade, created_at
		FROM users WHERE nickname = ? AND deleted_at IS NULL LIMIT 1
	`, name))
	if err != nil {
		return domain.AccountProfile{}, err
	}
	profile := domain.AccountProfile{User: user, ChoiceProfile: defaultChoiceProfile()}
	var profileJSON string
	if err := r.db.QueryRowContext(ctx, `SELECT bio, choice_profile FROM user_profiles WHERE user_id = ?`, user.ID).Scan(&profile.Bio, &profileJSON); err != nil && !errors.Is(err, sql.ErrNoRows) {
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
		if err := r.db.QueryRowContext(ctx, item.query, item.arg).Scan(item.dest); err != nil {
			return domain.AccountProfile{}, err
		}
	}
	profile.Posts, err = r.ListPosts(ctx, viewerID, domain.FeedFilter{AuthorName: user.Nickname, Sort: domain.SortLatest, Limit: 50})
	if err != nil {
		return domain.AccountProfile{}, err
	}
	profile.Comments, err = r.listProfileComments(ctx, user.ID)
	if err != nil {
		return domain.AccountProfile{}, err
	}
	if isOwner {
		profile.Favorites, err = r.listFavoritePosts(ctx, user.ID)
		if err != nil {
			return domain.AccountProfile{}, err
		}
	} else {
		profile.User.Email = ""
	}
	return profile, nil
}

func (r *ForumRepository) UpdateAccountProfile(ctx context.Context, userID int64, input domain.UpdateProfileInput) (domain.AccountProfile, error) {
	now := nowString()
	_, err := r.db.ExecContext(ctx, `
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
	return r.GetAccountProfile(ctx, &userID, user.Nickname)
}

func (r *ForumRepository) ListNotifications(ctx context.Context, userID int64) ([]domain.Notification, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT n.id, n.type, n.title, n.summary, n.target_url, COALESCE(u.nickname, ''), n.created_at, n.read_at
		FROM notifications n
		LEFT JOIN users u ON u.id = n.actor_user_id
		WHERE n.recipient_user_id = ?
		ORDER BY n.created_at DESC, n.id DESC
		LIMIT 100
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.Notification, 0)
	for rows.Next() {
		var item domain.Notification
		var createdAt string
		var readAt sql.NullString
		if err := rows.Scan(&item.ID, &item.Type, &item.Title, &item.Summary, &item.TargetURL, &item.ActorName, &createdAt, &readAt); err != nil {
			return nil, err
		}
		item.CreatedAt = parseTime(createdAt)
		if readAt.Valid {
			value := parseTime(readAt.String)
			item.ReadAt = &value
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *ForumRepository) MarkNotificationRead(ctx context.Context, userID int64, notificationID *int64) error {
	query := `UPDATE notifications SET read_at = COALESCE(read_at, ?) WHERE recipient_user_id = ?`
	args := []any{nowString(), userID}
	if notificationID != nil {
		query += ` AND id = ?`
		args = append(args, *notificationID)
	}
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *ForumRepository) togglePostRelation(ctx context.Context, table string, counter string, userID int64, postID int64) (domain.ToggleResult, error) {
	if _, err := r.fetchPostByID(ctx, postID); err != nil {
		return domain.ToggleResult{}, err
	}
	result, err := r.db.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE user_id = ? AND post_id = ?`, table), userID, postID)
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
		if _, err := r.db.ExecContext(ctx,
			fmt.Sprintf(`INSERT INTO %s (user_id, post_id, created_at) VALUES (?, ?, ?)`, table),
			userID,
			postID,
			nowString(),
		); err != nil {
			return domain.ToggleResult{}, err
		}
		active = true
		delta = 1
	}

	if _, err := r.db.ExecContext(ctx,
		fmt.Sprintf(`UPDATE posts SET %s = MAX(%s + ?, 0), updated_at = ? WHERE id = ?`, counter, counter),
		delta,
		nowString(),
		postID,
	); err != nil {
		return domain.ToggleResult{}, err
	}

	var count int
	if err := r.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT %s FROM posts WHERE id = ?`, counter), postID).Scan(&count); err != nil {
		return domain.ToggleResult{}, err
	}
	return domain.ToggleResult{Active: active, Count: count}, nil
}

func (r *ForumRepository) createNotification(ctx context.Context, recipientID int64, actorID *int64, notificationType string, title string, summary string, targetURL string) error {
	_, err := r.db.ExecContext(ctx, `
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
	if err := r.db.QueryRowContext(ctx, `SELECT nickname FROM users WHERE id = ?`, actorID).Scan(&actorName); err != nil {
		return err
	}
	return r.createNotification(ctx, *post.UserID, &actorID, notificationType, actorName+" "+action, post.Title, fmt.Sprintf("/posts/%d", postID))
}

func (r *ForumRepository) listProfileComments(ctx context.Context, userID int64) ([]domain.ProfileComment, error) {
	rows, err := r.db.QueryContext(ctx, `
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
	rows, err := r.db.QueryContext(ctx, `
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
	return scanPost(r.db.QueryRowContext(ctx, `
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
		rows, err := r.db.QueryContext(ctx, query, *viewerID)
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

	rows, err := r.db.QueryContext(ctx, `SELECT author_name FROM follows WHERE follower_id = ?`, *viewerID)
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
	var updatedAt string
	err := scanner.Scan(
		&insight.ID,
		&insight.Combination,
		&insight.Trend,
		&insight.Heat,
		&insight.MatchRate,
		&insight.Advice,
		&insight.Details,
		&updatedAt,
	)
	if err != nil {
		return domain.SubjectInsight{}, err
	}
	insight.UpdatedAt = parseTime(updatedAt)
	return insight, nil
}

func scanTopic(scanner topicScanner) (domain.Topic, error) {
	var topic domain.Topic
	var createdAt string
	err := scanner.Scan(
		&topic.ID,
		&topic.Slug,
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
	var createdAt string
	err := scanner.Scan(
		&user.ID,
		&user.Email,
		&user.Nickname,
		&user.Role,
		&user.Province,
		&user.Grade,
		&createdAt,
	)
	if err != nil {
		return domain.User{}, err
	}
	user.CreatedAt = parseTime(createdAt)
	return user, nil
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
