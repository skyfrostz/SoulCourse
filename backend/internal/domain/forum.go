package domain

import "time"

type SubjectTrack string
type Subject string
type PostCategory string
type FeedSort string

const (
	TrackPhysics SubjectTrack = "physics"
	TrackHistory SubjectTrack = "history"

	SubjectChemistry Subject = "chemistry"
	SubjectBiology   Subject = "biology"
	SubjectPolitics  Subject = "politics"
	SubjectGeography Subject = "geography"

	CategoryExperience PostCategory = "experience"
	CategoryQuestion   PostCategory = "question"
	CategoryData       PostCategory = "data"

	SortRecommended FeedSort = "recommended"
	SortLatest      FeedSort = "latest"
	SortHot         FeedSort = "hot"
)

type Post struct {
	ID              int64        `json:"id"`
	UserID          *int64       `json:"userId,omitempty"`
	AuthorName      string       `json:"authorName"`
	AuthorRole      string       `json:"authorRole"`
	Title           string       `json:"title"`
	Content         string       `json:"content"`
	ImageURLs       []string     `json:"imageUrls"`
	Tags            []string     `json:"tags"`
	Track           SubjectTrack `json:"track"`
	Electives       []Subject    `json:"electives"`
	Category        PostCategory `json:"category"`
	Grade           string       `json:"grade"`
	Province        string       `json:"province"`
	LikesCount      int          `json:"likesCount"`
	CommentsCount   int          `json:"commentsCount"`
	FavoritesCount  int          `json:"favoritesCount"`
	ViewerLiked     bool         `json:"viewerLiked"`
	ViewerFavorited bool         `json:"viewerFavorited"`
	ViewerFollowing bool         `json:"viewerFollowing"`
	CreatedAt       time.Time    `json:"createdAt"`
	UpdatedAt       time.Time    `json:"updatedAt"`
}

type Comment struct {
	ID        int64     `json:"id"`
	PostID    int64     `json:"postId"`
	UserID    *int64    `json:"userId,omitempty"`
	Author    string    `json:"author"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type FeedFilter struct {
	Track    SubjectTrack
	Subject  Subject
	Subjects []Subject
	Category PostCategory
	Province string
	Keyword  string
	Sort     FeedSort
	Limit    int
	Offset   int
}

type CreatePostInput struct {
	Title     string       `json:"title" binding:"required,min=4,max=80"`
	Content   string       `json:"content" binding:"required,min=10,max=4000"`
	ImageURLs []string     `json:"imageUrls" binding:"omitempty,max=9,dive,max=2000000"`
	Tags      []string     `json:"tags" binding:"omitempty,max=8,dive,max=20"`
	Track     SubjectTrack `json:"track" binding:"required,oneof=physics history"`
	Electives []Subject    `json:"electives" binding:"required,len=2,dive,oneof=chemistry biology politics geography"`
	Category  PostCategory `json:"category" binding:"required,oneof=experience question data"`
	Grade     string       `json:"grade" binding:"required,max=20"`
	Province  string       `json:"province" binding:"required,max=30"`
}

type CreateCommentInput struct {
	Content string `json:"content" binding:"required,min=2,max=1000"`
}

type SubjectInsight struct {
	ID          int64     `json:"id"`
	Combination string    `json:"combination"`
	Trend       string    `json:"trend"`
	Heat        int       `json:"heat"`
	MatchRate   float64   `json:"matchRate"`
	Advice      string    `json:"advice"`
	Details     string    `json:"details"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Nickname  string    `json:"nickname"`
	Role      string    `json:"role"`
	Province  string    `json:"province"`
	Grade     string    `json:"grade"`
	CreatedAt time.Time `json:"createdAt"`
}

type RegisterInput struct {
	Email            string `json:"email" binding:"required,email,max=120"`
	Password         string `json:"password" binding:"required,min=6,max=72"`
	VerificationCode string `json:"verificationCode" binding:"required,len=6,numeric"`
	Nickname         string `json:"nickname" binding:"required,min=1,max=40"`
	Role             string `json:"role" binding:"required,oneof=student parent teacher counselor"`
	Province         string `json:"province" binding:"required,max=30"`
	Grade            string `json:"grade" binding:"required,max=20"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type EmailVerificationCodeInput struct {
	Email string `json:"email" binding:"required,email,max=120"`
}

type EmailVerificationCodeResult struct {
	Email            string `json:"email"`
	ExpiresInSeconds int    `json:"expiresInSeconds"`
	DebugCode        string `json:"debugCode,omitempty"`
}

type AuthSession struct {
	User  User   `json:"user"`
	Token string `json:"token"`
}

type Topic struct {
	ID         int64     `json:"id"`
	Slug       string    `json:"slug"`
	Title      string    `json:"title"`
	Summary    string    `json:"summary"`
	ViewsCount int       `json:"viewsCount"`
	PostsCount int       `json:"postsCount"`
	CreatedAt  time.Time `json:"createdAt"`
}

type TopicDetail struct {
	Topic Topic  `json:"topic"`
	Posts []Post `json:"posts"`
}

type ToggleResult struct {
	Active bool `json:"active"`
	Count  int  `json:"count"`
}

type ChoiceAdviceInput struct {
	Profile  map[string]any `json:"profile" binding:"required"`
	Question string         `json:"question" binding:"omitempty,max=500"`
}

type ChoiceAdvice struct {
	Summary          string   `json:"summary"`
	Risks            []string `json:"risks"`
	Actions          []string `json:"actions"`
	QuerySuggestions []string `json:"querySuggestions"`
	Source           string   `json:"source"`
}

func TrackLabel(track SubjectTrack) string {
	switch track {
	case TrackPhysics:
		return "物理方向"
	case TrackHistory:
		return "历史方向"
	default:
		return string(track)
	}
}

func SubjectLabel(subject Subject) string {
	switch subject {
	case SubjectChemistry:
		return "化学"
	case SubjectBiology:
		return "生物"
	case SubjectPolitics:
		return "政治"
	case SubjectGeography:
		return "地理"
	default:
		return string(subject)
	}
}
