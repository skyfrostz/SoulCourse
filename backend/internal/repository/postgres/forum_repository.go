package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"subject-choice-forum/backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ForumRepository struct {
	db *pgxpool.Pool
}

func NewForumRepository(db *pgxpool.Pool) *ForumRepository {
	return &ForumRepository{db: db}
}

func (r *ForumRepository) ListPosts(ctx context.Context, viewerID *int64, filter domain.FeedFilter) ([]domain.Post, error) {
	args := []any{}
	clauses := []string{"deleted_at IS NULL"}

	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	if filter.Track != "" {
		clauses = append(clauses, "track = "+addArg(filter.Track))
	}
	if filter.Subject != "" {
		clauses = append(clauses, addArg(filter.Subject)+" = ANY(electives)")
	}
	for _, subject := range filter.Subjects {
		if subject != "" {
			clauses = append(clauses, addArg(subject)+" = ANY(electives)")
		}
	}
	if filter.Category != "" {
		clauses = append(clauses, "category = "+addArg(filter.Category))
	}
	if filter.Province != "" {
		clauses = append(clauses, "province = "+addArg(filter.Province))
	}
	if filter.Keyword != "" {
		keywordArg := addArg("%" + filter.Keyword + "%")
		clauses = append(clauses, "(title ILIKE "+keywordArg+" OR content ILIKE "+keywordArg+")")
	}

	limitArg := addArg(filter.Limit)
	offsetArg := addArg(filter.Offset)
	viewerArg := addArg(viewerID)
	orderBy := `
		(
		  LEAST(likes_count, 300) * 0.8 +
		  LEAST(comments_count, 80) * 4 +
		  LEAST(favorites_count, 120) * 3 +
		  CASE WHEN author_role IN ('teacher', 'counselor') THEN 45 ELSE 0 END +
		  CASE WHEN likes_count < 150 THEN 65 ELSE 0 END
		) DESC,
		created_at DESC`
	switch filter.Sort {
	case domain.SortLatest:
		orderBy = "created_at DESC"
	case domain.SortHot:
		orderBy = "likes_count + comments_count * 4 DESC, updated_at DESC"
	}

	query := `
		SELECT id, user_id, author_name, author_role, title, content, image_urls, tags, track, electives, category, grade,
		       province, likes_count, comments_count, favorites_count,
		       CASE WHEN ` + viewerArg + `::bigint IS NULL THEN false ELSE EXISTS(SELECT 1 FROM post_likes WHERE post_id = posts.id AND user_id = ` + viewerArg + `::bigint) END,
		       CASE WHEN ` + viewerArg + `::bigint IS NULL THEN false ELSE EXISTS(SELECT 1 FROM post_favorites WHERE post_id = posts.id AND user_id = ` + viewerArg + `::bigint) END,
		       CASE WHEN ` + viewerArg + `::bigint IS NULL THEN false ELSE EXISTS(SELECT 1 FROM follows WHERE author_name = posts.author_name AND follower_id = ` + viewerArg + `::bigint) END,
		       created_at, updated_at
		FROM posts
		WHERE ` + strings.Join(clauses, " AND ") + `
		ORDER BY ` + orderBy + `
		LIMIT ` + limitArg + ` OFFSET ` + offsetArg

	rows, err := r.db.Query(ctx, query, args...)
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

func (r *ForumRepository) GetPost(ctx context.Context, viewerID *int64, id int64) (domain.Post, []domain.Comment, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.Post{}, nil, err
	}
	defer tx.Rollback(ctx)

	postRow := tx.QueryRow(ctx, `
		SELECT id, user_id, author_name, author_role, title, content, image_urls, tags, track, electives, category, grade,
		       province, likes_count, comments_count, favorites_count,
		       CASE WHEN $2::bigint IS NULL THEN false ELSE EXISTS(SELECT 1 FROM post_likes WHERE post_id = posts.id AND user_id = $2::bigint) END,
		       CASE WHEN $2::bigint IS NULL THEN false ELSE EXISTS(SELECT 1 FROM post_favorites WHERE post_id = posts.id AND user_id = $2::bigint) END,
		       CASE WHEN $2::bigint IS NULL THEN false ELSE EXISTS(SELECT 1 FROM follows WHERE author_name = posts.author_name AND follower_id = $2::bigint) END,
		       created_at, updated_at
		FROM posts
		WHERE id = $1 AND deleted_at IS NULL
	`, id, viewerID)

	post, err := scanPost(postRow)
	if err != nil {
		return domain.Post{}, nil, err
	}

	rows, err := tx.Query(ctx, `
		SELECT id, post_id, user_id, author, role, content, created_at
		FROM comments
		WHERE post_id = $1 AND deleted_at IS NULL
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
	if err := rows.Err(); err != nil {
		return domain.Post{}, nil, err
	}

	return post, comments, tx.Commit(ctx)
}

func (r *ForumRepository) CreatePost(ctx context.Context, user domain.User, input domain.CreatePostInput) (domain.Post, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO posts (user_id, author_name, author_role, title, content, image_urls, tags, track, electives, category, grade, province)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, user_id, author_name, author_role, title, content, image_urls, tags, track, electives, category, grade,
		          province, likes_count, comments_count, favorites_count, false, false, false, created_at, updated_at
	`, user.ID, user.Nickname, user.Role, input.Title, input.Content, input.ImageURLs, input.Tags, input.Track, subjectStrings(input.Electives), input.Category, user.Grade, user.Province)

	return scanPost(row)
}

func (r *ForumRepository) CreateComment(ctx context.Context, user domain.User, postID int64, input domain.CreateCommentInput) (domain.Comment, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.Comment{}, err
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM posts WHERE id = $1 AND deleted_at IS NULL)`, postID).Scan(&exists); err != nil {
		return domain.Comment{}, err
	}
	if !exists {
		return domain.Comment{}, pgx.ErrNoRows
	}

	comment, err := scanComment(tx.QueryRow(ctx, `
		INSERT INTO comments (post_id, user_id, author, role, content)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, post_id, user_id, author, role, content, created_at
	`, postID, user.ID, user.Nickname, user.Role, input.Content))
	if err != nil {
		return domain.Comment{}, err
	}

	if _, err := tx.Exec(ctx, `UPDATE posts SET comments_count = comments_count + 1, updated_at = now() WHERE id = $1`, postID); err != nil {
		return domain.Comment{}, err
	}

	return comment, tx.Commit(ctx)
}

func (r *ForumRepository) ListInsights(ctx context.Context) ([]domain.SubjectInsight, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, combination, trend, heat, match_rate, advice, details, updated_at
		FROM subject_insights
		ORDER BY heat DESC
		LIMIT 8
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	insights := make([]domain.SubjectInsight, 0)
	for rows.Next() {
		insight, err := scanInsight(rows)
		if err != nil {
			return nil, err
		}
		insights = append(insights, insight)
	}
	return insights, rows.Err()
}

func (r *ForumRepository) GetInsight(ctx context.Context, id int64) (domain.SubjectInsight, error) {
	return scanInsight(r.db.QueryRow(ctx, `
		SELECT id, combination, trend, heat, match_rate, advice, details, updated_at
		FROM subject_insights
		WHERE id = $1
	`, id))
}

func (r *ForumRepository) ListTopics(ctx context.Context) ([]domain.Topic, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, slug, title, summary, views_count, posts_count, created_at
		FROM topics
		ORDER BY views_count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	topics := make([]domain.Topic, 0)
	for rows.Next() {
		topic, err := scanTopic(rows)
		if err != nil {
			return nil, err
		}
		topics = append(topics, topic)
	}
	return topics, rows.Err()
}

func (r *ForumRepository) GetTopic(ctx context.Context, viewerID *int64, slug string) (domain.TopicDetail, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.TopicDetail{}, err
	}
	defer tx.Rollback(ctx)

	topic, err := scanTopic(tx.QueryRow(ctx, `
		UPDATE topics SET views_count = views_count + 1 WHERE slug = $1
		RETURNING id, slug, title, summary, views_count, posts_count, created_at
	`, slug))
	if err != nil {
		return domain.TopicDetail{}, err
	}

	posts, err := r.listTopicPosts(ctx, tx, viewerID, topic.ID)
	if err != nil {
		return domain.TopicDetail{}, err
	}
	return domain.TopicDetail{Topic: topic, Posts: posts}, tx.Commit(ctx)
}

func (r *ForumRepository) CreateUser(ctx context.Context, input domain.RegisterInput, passwordHash string) (domain.User, error) {
	return scanUser(r.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, nickname, role, province, grade)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, email, nickname, role, province, grade, created_at
	`, input.Email, passwordHash, input.Nickname, input.Role, input.Province, input.Grade))
}

func (r *ForumRepository) GetUserByEmail(ctx context.Context, email string) (domain.User, string, error) {
	var user domain.User
	var passwordHash string
	err := r.db.QueryRow(ctx, `
		SELECT id, email, nickname, role, province, grade, created_at, password_hash
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`, email).Scan(&user.ID, &user.Email, &user.Nickname, &user.Role, &user.Province, &user.Grade, &user.CreatedAt, &passwordHash)
	return user, passwordHash, err
}

func (r *ForumRepository) GetUserByID(ctx context.Context, id int64) (domain.User, error) {
	return scanUser(r.db.QueryRow(ctx, `
		SELECT id, email, nickname, role, province, grade, created_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`, id))
}

func (r *ForumRepository) TogglePostLike(ctx context.Context, userID int64, postID int64) (domain.ToggleResult, error) {
	return r.togglePostRelation(ctx, "post_likes", "likes_count", userID, postID)
}

func (r *ForumRepository) TogglePostFavorite(ctx context.Context, userID int64, postID int64) (domain.ToggleResult, error) {
	return r.togglePostRelation(ctx, "post_favorites", "favorites_count", userID, postID)
}

func (r *ForumRepository) ToggleFollowAuthor(ctx context.Context, followerID int64, authorName string) (bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM posts WHERE author_name = $1 AND deleted_at IS NULL)`, authorName).Scan(&exists); err != nil {
		return false, err
	}
	if !exists {
		return false, pgx.ErrNoRows
	}

	tag, err := tx.Exec(ctx, `DELETE FROM follows WHERE follower_id = $1 AND author_name = $2`, followerID, authorName)
	if err != nil {
		return false, err
	}
	active := false
	if tag.RowsAffected() == 0 {
		if _, err := tx.Exec(ctx, `INSERT INTO follows (follower_id, author_name) VALUES ($1, $2) ON CONFLICT DO NOTHING`, followerID, authorName); err != nil {
			return false, err
		}
		active = true
	}
	return active, tx.Commit(ctx)
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
	var electives []string
	var imageURLs []string
	var tags []string
	var userID sql.NullInt64
	err := scanner.Scan(
		&post.ID,
		&userID,
		&post.AuthorName,
		&post.AuthorRole,
		&post.Title,
		&post.Content,
		&imageURLs,
		&tags,
		&post.Track,
		&electives,
		&post.Category,
		&post.Grade,
		&post.Province,
		&post.LikesCount,
		&post.CommentsCount,
		&post.FavoritesCount,
		&post.ViewerLiked,
		&post.ViewerFavorited,
		&post.ViewerFollowing,
		&post.CreatedAt,
		&post.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Post{}, err
		}
		return domain.Post{}, err
	}
	if userID.Valid {
		post.UserID = &userID.Int64
	}
	post.Electives = subjectsFromStrings(electives)
	post.ImageURLs = imageURLs
	post.Tags = tags
	return post, nil
}

func scanComment(scanner commentScanner) (domain.Comment, error) {
	var comment domain.Comment
	var userID sql.NullInt64
	err := scanner.Scan(
		&comment.ID,
		&comment.PostID,
		&userID,
		&comment.Author,
		&comment.Role,
		&comment.Content,
		&comment.CreatedAt,
	)
	if userID.Valid {
		comment.UserID = &userID.Int64
	}
	return comment, err
}

func scanInsight(scanner insightScanner) (domain.SubjectInsight, error) {
	var insight domain.SubjectInsight
	err := scanner.Scan(
		&insight.ID,
		&insight.Combination,
		&insight.Trend,
		&insight.Heat,
		&insight.MatchRate,
		&insight.Advice,
		&insight.Details,
		&insight.UpdatedAt,
	)
	return insight, err
}

func scanTopic(scanner topicScanner) (domain.Topic, error) {
	var topic domain.Topic
	err := scanner.Scan(
		&topic.ID,
		&topic.Slug,
		&topic.Title,
		&topic.Summary,
		&topic.ViewsCount,
		&topic.PostsCount,
		&topic.CreatedAt,
	)
	return topic, err
}

func scanUser(scanner userScanner) (domain.User, error) {
	var user domain.User
	err := scanner.Scan(
		&user.ID,
		&user.Email,
		&user.Nickname,
		&user.Role,
		&user.Province,
		&user.Grade,
		&user.CreatedAt,
	)
	return user, err
}

func (r *ForumRepository) togglePostRelation(ctx context.Context, table string, counter string, userID int64, postID int64) (domain.ToggleResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.ToggleResult{}, err
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM posts WHERE id = $1 AND deleted_at IS NULL)`, postID).Scan(&exists); err != nil {
		return domain.ToggleResult{}, err
	}
	if !exists {
		return domain.ToggleResult{}, pgx.ErrNoRows
	}

	deleteSQL := fmt.Sprintf(`DELETE FROM %s WHERE user_id = $1 AND post_id = $2`, table)
	tag, err := tx.Exec(ctx, deleteSQL, userID, postID)
	if err != nil {
		return domain.ToggleResult{}, err
	}

	active := false
	delta := -1
	if tag.RowsAffected() == 0 {
		insertSQL := fmt.Sprintf(`INSERT INTO %s (user_id, post_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, table)
		if _, err := tx.Exec(ctx, insertSQL, userID, postID); err != nil {
			return domain.ToggleResult{}, err
		}
		active = true
		delta = 1
	}

	updateSQL := fmt.Sprintf(`UPDATE posts SET %s = GREATEST(%s + $1, 0), updated_at = now() WHERE id = $2 RETURNING %s`, counter, counter, counter)
	var count int
	if err := tx.QueryRow(ctx, updateSQL, delta, postID).Scan(&count); err != nil {
		return domain.ToggleResult{}, err
	}

	return domain.ToggleResult{Active: active, Count: count}, tx.Commit(ctx)
}

func (r *ForumRepository) listTopicPosts(ctx context.Context, tx pgx.Tx, viewerID *int64, topicID int64) ([]domain.Post, error) {
	rows, err := tx.Query(ctx, `
		SELECT p.id, p.user_id, p.author_name, p.author_role, p.title, p.content, p.image_urls, p.tags, p.track, p.electives, p.category, p.grade,
		       p.province, p.likes_count, p.comments_count, p.favorites_count,
		       CASE WHEN $2::bigint IS NULL THEN false ELSE EXISTS(SELECT 1 FROM post_likes WHERE post_id = p.id AND user_id = $2::bigint) END,
		       CASE WHEN $2::bigint IS NULL THEN false ELSE EXISTS(SELECT 1 FROM post_favorites WHERE post_id = p.id AND user_id = $2::bigint) END,
		       CASE WHEN $2::bigint IS NULL THEN false ELSE EXISTS(SELECT 1 FROM follows WHERE author_name = p.author_name AND follower_id = $2::bigint) END,
		       p.created_at, p.updated_at
		FROM posts p
		JOIN topic_posts tp ON tp.post_id = p.id
		WHERE tp.topic_id = $1 AND p.deleted_at IS NULL
		ORDER BY p.created_at DESC
	`, topicID, viewerID)
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
