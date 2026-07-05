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
	posts, err := r.fetchPosts(ctx)
	if err != nil {
		return nil, err
	}

	liked, favorited, followed, err := r.viewerState(ctx, viewerID)
	if err != nil {
		return nil, err
	}

	filtered := make([]domain.Post, 0, len(posts))
	for _, post := range posts {
		if !matchesPostFilter(post, filter) {
			continue
		}
		post.ViewerLiked = liked[post.ID]
		post.ViewerFavorited = favorited[post.ID]
		post.ViewerFollowing = followed[post.AuthorName]
		filtered = append(filtered, post)
	}

	sortPosts(filtered, filter.Sort)

	start := minInt(filter.Offset, len(filtered))
	end := minInt(start+filter.Limit, len(filtered))
	return filtered[start:end], nil
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
	post.ViewerFollowing = followed[post.AuthorName]

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
	if _, err := r.fetchPostByID(ctx, postID); err != nil {
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
		       p.category, p.grade, p.province, p.likes_count, p.comments_count, p.favorites_count, p.created_at, p.updated_at
		FROM posts p
		JOIN topic_posts tp ON tp.post_id = p.id
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
		post.ViewerFollowing = followed[post.AuthorName]
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

func (r *ForumRepository) ConsumeEmailVerificationCode(ctx context.Context, email string, codeHash string) error {
	now := nowString()
	result, err := r.db.ExecContext(ctx, `
		UPDATE email_verification_codes
		SET used_at = ?
		WHERE id = (
			SELECT id
			FROM email_verification_codes
			WHERE lower(email) = lower(?)
			  AND code_hash = ?
			  AND used_at IS NULL
			  AND expires_at > ?
			ORDER BY created_at DESC
			LIMIT 1
		)
	`, now, email, codeHash, now)
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

func (r *ForumRepository) TogglePostLike(ctx context.Context, userID int64, postID int64) (domain.ToggleResult, error) {
	return r.togglePostRelation(ctx, "post_likes", "likes_count", userID, postID)
}

func (r *ForumRepository) TogglePostFavorite(ctx context.Context, userID int64, postID int64) (domain.ToggleResult, error) {
	return r.togglePostRelation(ctx, "post_favorites", "favorites_count", userID, postID)
}

func (r *ForumRepository) ToggleFollowAuthor(ctx context.Context, followerID int64, authorName string) (bool, error) {
	var exists int
	if err := r.db.QueryRowContext(ctx, `
		SELECT 1
		FROM posts
		WHERE author_name = ? AND deleted_at IS NULL
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
	return true, nil
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

func (r *ForumRepository) fetchPosts(ctx context.Context) ([]domain.Post, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, author_name, author_role, title, content, image_urls, tags, track, electives,
		       category, grade, province, likes_count, comments_count, favorites_count, created_at, updated_at
		FROM posts
		WHERE deleted_at IS NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := make([]domain.Post, 0)
	for rows.Next() {
		post, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, rows.Err()
}

func (r *ForumRepository) fetchPostByID(ctx context.Context, id int64) (domain.Post, error) {
	return scanPost(r.db.QueryRowContext(ctx, `
		SELECT id, user_id, author_name, author_role, title, content, image_urls, tags, track, electives,
		       category, grade, province, likes_count, comments_count, favorites_count, created_at, updated_at
		FROM posts
		WHERE id = ? AND deleted_at IS NULL
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

func matchesPostFilter(post domain.Post, filter domain.FeedFilter) bool {
	if filter.Track != "" && post.Track != filter.Track {
		return false
	}
	if filter.Category != "" && post.Category != filter.Category {
		return false
	}
	if strings.TrimSpace(filter.Province) != "" && post.Province != filter.Province {
		return false
	}
	subjects := append([]domain.Subject{}, filter.Subjects...)
	if filter.Subject != "" {
		subjects = append(subjects, filter.Subject)
	}
	for _, subject := range subjects {
		if subject == "" {
			continue
		}
		found := false
		for _, elective := range post.Electives {
			if elective == subject {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if keyword := strings.ToLower(strings.TrimSpace(filter.Keyword)); keyword != "" {
		text := strings.ToLower(strings.Join([]string{
			post.Title,
			post.Content,
			post.AuthorName,
			post.Province,
			post.Grade,
			strings.Join(post.Tags, ","),
		}, " "))
		if !strings.Contains(text, keyword) {
			return false
		}
	}
	return true
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
