package main

import (
	"net/http"
	"testing"
)

type getTagsResponse struct {
	Tags []string `json:"tags"`
}

func TestGetTagsHandler(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)

	// Register Alice and Bob users
	registerUser(t, ts, "alice", "alice@example.com", "password123")
	registerUser(t, ts, "bob", "bob@example.com", "password123")
	aliceToken := loginUser(t, ts, "alice@example.com", "password123")
	bobToken := loginUser(t, ts, "bob@example.com", "password123")

	// Create articles by Alice and Bob
	createArticle(t, ts, aliceToken, "Alice's Go Tutorial", "Learn Go programming", "This is a comprehensive guide to Go programming language.", []string{"golang", "backend"})
	createArticle(t, ts, aliceToken, "Frontend Development", "Modern web development", "Building modern web applications with latest technologies.", []string{"frontend", "javascript"})
	createArticle(t, ts, bobToken, "Backend Architecture", "Scalable backend systems", "Designing scalable and maintainable backend architectures.", []string{"golang", "backend", "testing"})
	createArticle(t, ts, bobToken, "Web Development Best Practices", "Best practices for web dev", "Essential practices for modern web development.", []string{"frontend", "backend"})

	testcases := []handlerTestcase{
		{
			name:                   "Get all tags successfully",
			requestMethodType:      http.MethodGet,
			requestUrlPath:         "/tags",
			wantResponseStatusCode: http.StatusOK,
			wantResponse: getTagsResponse{
				Tags: []string{"backend", "frontend", "golang", "javascript", "testing"},
			},
		},
		{
			name:                   "Method not allowed",
			requestMethodType:      http.MethodPost,
			requestUrlPath:         "/tags",
			wantResponseStatusCode: http.StatusMethodNotAllowed,
		},
	}

	testHandler(t, ts, testcases...)
}

func TestGetTagsHandler_NoTags(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)

	testcases := []handlerTestcase{
		{
			name:                   "Get tags when none exist",
			requestMethodType:      http.MethodGet,
			requestUrlPath:         "/tags",
			wantResponseStatusCode: http.StatusOK,
			wantResponse: getTagsResponse{
				Tags: []string{},
			},
		},
	}
	testHandler(t, ts, testcases...)
}
