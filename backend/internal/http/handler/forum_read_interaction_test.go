package handler

import (
	"net/http"
	"strings"
	"testing"
)

func TestForumHandlerReadAndInteractionBranches(t *testing.T) {
	h := newForumHandlerHarness(t)
	taxonomy := h.request(http.MethodGet, "/taxonomy", "", "")
	if taxonomy.Code != http.StatusOK || !strings.Contains(taxonomy.Body.String(), `"tracks"`) || !strings.Contains(taxonomy.Body.String(), `"topicTags"`) {
		t.Fatalf("taxonomy status=%d body=%s", taxonomy.Code, taxonomy.Body.String())
	}

	postID := createForumHandlerPost(t, h, "owner")
	posts := h.request(http.MethodGet, "/posts?limit=10&subjects=chemistry,biology&sort=latest", "", "")
	if posts.Code != http.StatusOK || !strings.Contains(posts.Body.String(), "待举报公测帖子") {
		t.Fatalf("posts status=%d body=%s", posts.Code, posts.Body.String())
	}

	detail := h.request(http.MethodGet, pathID("/posts/", postID), "peer", "")
	if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), `"comments"`) || !strings.Contains(detail.Body.String(), "待举报公测帖子") {
		t.Fatalf("post detail status=%d body=%s", detail.Code, detail.Body.String())
	}

	invalidDetail := h.request(http.MethodGet, "/posts/nope", "peer", "")
	assertHandlerError(t, invalidDetail, http.StatusBadRequest, "invalid_id")
	missingDetail := h.request(http.MethodGet, "/posts/999999", "peer", "")
	assertHandlerError(t, missingDetail, http.StatusNotFound, "not_found")

	comment := h.request(http.MethodPost, pathID("/posts/", postID)+"/comments", "peer", `{"content":"这条评论用于验证公测互动"}`)
	if comment.Code != http.StatusCreated || !strings.Contains(comment.Body.String(), "这条评论用于验证公测互动") {
		t.Fatalf("comment status=%d body=%s", comment.Code, comment.Body.String())
	}
	assertRowCount(t, h.db, "SELECT COUNT(*) FROM comments WHERE post_id = ? AND deleted_at IS NULL", 1, postID)
	invalidComment := h.request(http.MethodPost, pathID("/posts/", postID)+"/comments", "peer", `{"content":"   "}`)
	assertHandlerError(t, invalidComment, http.StatusBadRequest, "invalid_comment")

	like := h.request(http.MethodPost, pathID("/posts/", postID)+"/like", "peer", "")
	if like.Code != http.StatusOK || !strings.Contains(like.Body.String(), `"active":true`) {
		t.Fatalf("like status=%d body=%s", like.Code, like.Body.String())
	}
	favorite := h.request(http.MethodPost, pathID("/posts/", postID)+"/favorite", "peer", "")
	if favorite.Code != http.StatusOK || !strings.Contains(favorite.Body.String(), `"active":true`) {
		t.Fatalf("favorite status=%d body=%s", favorite.Code, favorite.Body.String())
	}
	assertRowCount(t, h.db, "SELECT COUNT(*) FROM post_likes WHERE post_id = ? AND user_id = ?", 1, postID, h.users["peer"].ID)
	assertRowCount(t, h.db, "SELECT COUNT(*) FROM post_favorites WHERE post_id = ? AND user_id = ?", 1, postID, h.users["peer"].ID)
	likeOff := h.request(http.MethodPost, pathID("/posts/", postID)+"/like", "peer", "")
	if likeOff.Code != http.StatusOK || !strings.Contains(likeOff.Body.String(), `"active":false`) {
		t.Fatalf("like off status=%d body=%s", likeOff.Code, likeOff.Body.String())
	}
	favoriteOff := h.request(http.MethodPost, pathID("/posts/", postID)+"/favorite", "peer", "")
	if favoriteOff.Code != http.StatusOK || !strings.Contains(favoriteOff.Body.String(), `"active":false`) {
		t.Fatalf("favorite off status=%d body=%s", favoriteOff.Code, favoriteOff.Body.String())
	}
	follow := h.request(http.MethodPost, "/authors/发帖学生/follow", "peer", "")
	if follow.Code != http.StatusOK || !strings.Contains(follow.Body.String(), `"active":true`) {
		t.Fatalf("follow status=%d body=%s", follow.Code, follow.Body.String())
	}
	followOff := h.request(http.MethodPost, "/authors/发帖学生/follow", "peer", "")
	if followOff.Code != http.StatusOK || !strings.Contains(followOff.Body.String(), `"active":false`) {
		t.Fatalf("follow off status=%d body=%s", followOff.Code, followOff.Body.String())
	}
	missingFollow := h.request(http.MethodPost, "/authors/不存在/follow", "peer", "")
	assertHandlerError(t, missingFollow, http.StatusNotFound, "not_found")
	missingInsight := h.request(http.MethodGet, "/insights/999999", "", "")
	assertHandlerError(t, missingInsight, http.StatusNotFound, "not_found")
	missingTopic := h.request(http.MethodGet, "/topics/not-found", "", "")
	assertHandlerError(t, missingTopic, http.StatusNotFound, "not_found")
	assertRowCount(t, h.db, "SELECT COUNT(*) FROM post_likes WHERE post_id = ? AND user_id = ?", 0, postID, h.users["peer"].ID)
	assertRowCount(t, h.db, "SELECT COUNT(*) FROM post_favorites WHERE post_id = ? AND user_id = ?", 0, postID, h.users["peer"].ID)

	for _, path := range []string{"/insights", "/topics"} {
		response := h.request(http.MethodGet, path, "", "")
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"data"`) {
			t.Fatalf("%s status=%d body=%s", path, response.Code, response.Body.String())
		}
	}
	if err := h.db.Close(); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/notifications/read-all", "/insights", "/topics"} {
		response := h.request(http.MethodGet, path, "peer", "")
		if path == "/notifications/read-all" {
			response = h.request(http.MethodPost, path, "peer", "")
		}
		if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), `"internal_error"`) {
			t.Fatalf("closed database %s status=%d body=%s", path, response.Code, response.Body.String())
		}
	}
}
