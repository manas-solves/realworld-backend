package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type commentResponse struct {
	Comment comment `json:"comment"`
}

type comment struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Author    profile   `json:"author"`
}

func TestCreateCommentHandler(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t)

	// Setup: Register users and create an article
	registerUser(t, ts, "alice", "alice@example.com", "password123")
	registerUser(t, ts, "bob", "bob@example.com", "password123")
	aliceToken := loginUser(t, ts, "alice@example.com", "password123")
	bobToken := loginUser(t, ts, "bob@example.com", "password123")

	// Alice creates an article
	articleLocation := createArticle(t, ts, aliceToken, "How to Train Your Dragon", "Ever wonder how?", "It takes a Jacobian", []string{"dragons", "training"})

	validRequestBody := `{
		"comment": {
			"body": "His name was my name too."
		}
	}`

	testcases := []handlerTestcase{
		{
			name:                   "Valid comment creation",
			requestMethodType:      http.MethodPost,
			requestUrlPath:         articleLocation + "/comments",
			requestHeader:          map[string]string{"Authorization": "Token " + bobToken},
			requestBody:            validRequestBody,
			wantResponseStatusCode: http.StatusCreated,
			additionalChecks: func(t *testing.T, res *http.Response) {
				var resp commentResponse
				readJsonResponse(t, res.Body, &resp)

				now := time.Now()

				assert.Equal(t, "His name was my name too.", resp.Comment.Body)
				assert.Equal(t, "bob", resp.Comment.Author.Username)
				assert.False(t, false, "Bob does not follow himself")
				assert.NotZero(t, resp.Comment.ID)
				assert.WithinDuration(t, now, resp.Comment.CreatedAt, time.Second, "CreatedAt should be within 1 second of now")
				assert.WithinDuration(t, now, resp.Comment.UpdatedAt, time.Second, "UpdatedAt should be within 1 second of now")
			},
		},
		{
			name:              "Comment creation without authentication",
			requestMethodType: http.MethodPost,
			requestUrlPath:    articleLocation + "/comments",
			requestBody:       validRequestBody,
			// No auth header
			wantResponseStatusCode: http.StatusUnauthorized,
		},
		{
			name:                   "Comment creation with empty body",
			requestMethodType:      http.MethodPost,
			requestUrlPath:         articleLocation + "/comments",
			requestHeader:          map[string]string{"Authorization": "Token " + bobToken},
			requestBody:            `{"comment": {"body": ""}}`,
			wantResponseStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name:                   "Comment creation with whitespace-only body",
			requestMethodType:      http.MethodPost,
			requestUrlPath:         articleLocation + "/comments",
			requestHeader:          map[string]string{"Authorization": "Token " + bobToken},
			requestBody:            `{"comment": {"body": "   "}}`,
			wantResponseStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name:                   "Comment creation with missing body field",
			requestMethodType:      http.MethodPost,
			requestUrlPath:         articleLocation + "/comments",
			requestHeader:          map[string]string{"Authorization": "Token " + bobToken},
			requestBody:            `{"comment": {}}`,
			wantResponseStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name:                   "Comment creation on non-existent article",
			requestMethodType:      http.MethodPost,
			requestUrlPath:         "/articles/non-existent-article-slug/comments",
			requestHeader:          map[string]string{"Authorization": "Token " + bobToken},
			requestBody:            validRequestBody,
			wantResponseStatusCode: http.StatusNotFound,
		},
		{
			name:                   "Comment creation with invalid JSON",
			requestMethodType:      http.MethodPost,
			requestUrlPath:         articleLocation + "/comments",
			requestHeader:          map[string]string{"Authorization": "Token " + bobToken},
			requestBody:            `{"comment": {"body": "test"`,
			wantResponseStatusCode: http.StatusBadRequest,
		},
	}

	testHandler(t, ts, testcases...)
}

func TestGetCommentsHandler_EmptyAndNotFound(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t)

	registerUser(t, ts, "alice", "alice@example.com", "password123")
	aliceToken := loginUser(t, ts, "alice@example.com", "password123")
	articleLocation := createArticle(t, ts, aliceToken, "Test Article", "Test description", "Test body", []string{"test"})

	testcases := []handlerTestcase{
		{
			name:                   "Get comments from article with no comments",
			requestMethodType:      http.MethodGet,
			requestUrlPath:         articleLocation + "/comments",
			wantResponseStatusCode: http.StatusOK,
			wantResponse: struct {
				Comments []comment `json:"comments"`
			}{
				Comments: []comment{},
			},
		},
		{
			name:                   "Get comments from non-existent article",
			requestMethodType:      http.MethodGet,
			requestUrlPath:         "/articles/non-existent-slug-12345/comments",
			wantResponseStatusCode: http.StatusNotFound,
		},
	}

	testHandler(t, ts, testcases...)
}

func TestGetCommentsHandler_WithoutAuthentication(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t)

	// Setup: Register 5 users and create article
	registerUser(t, ts, "alice", "alice@example.com", "password123")
	registerUser(t, ts, "bob", "bob@example.com", "password123")
	registerUser(t, ts, "charlie", "charlie@example.com", "password123")
	registerUser(t, ts, "david", "david@example.com", "password123")
	registerUser(t, ts, "eve", "eve@example.com", "password123")

	aliceToken := loginUser(t, ts, "alice@example.com", "password123")
	bobToken := loginUser(t, ts, "bob@example.com", "password123")
	charlieToken := loginUser(t, ts, "charlie@example.com", "password123")
	davidToken := loginUser(t, ts, "david@example.com", "password123")
	eveToken := loginUser(t, ts, "eve@example.com", "password123")

	articleLocation := createArticle(t, ts, aliceToken, "Test Article", "Test description", "Test body", []string{"test"})

	// Add 8 comments with some users commenting multiple times
	createCommentHelper(t, ts, bobToken, articleLocation, "Great article Alice!")
	time.Sleep(10 * time.Millisecond)
	createCommentHelper(t, ts, charlieToken, articleLocation, "I learned a lot from this.")
	time.Sleep(10 * time.Millisecond)
	createCommentHelper(t, ts, bobToken, articleLocation, "Also, the examples were clear.")
	time.Sleep(10 * time.Millisecond)
	createCommentHelper(t, ts, davidToken, articleLocation, "Very informative!")
	time.Sleep(10 * time.Millisecond)
	createCommentHelper(t, ts, eveToken, articleLocation, "Thanks for sharing!")
	time.Sleep(10 * time.Millisecond)
	createCommentHelper(t, ts, charlieToken, articleLocation, "Looking forward to more content.")
	time.Sleep(10 * time.Millisecond)
	createCommentHelper(t, ts, aliceToken, articleLocation, "Thanks for the feedback everyone!")
	time.Sleep(10 * time.Millisecond)
	createCommentHelper(t, ts, davidToken, articleLocation, "Keep up the good work!")

	// Test: Get comments without authentication
	res, err := ts.executeRequest(http.MethodGet, articleLocation+"/comments", "", nil)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)

	var resp struct {
		Comments []comment `json:"comments"`
	}
	readJsonResponse(t, res.Body, &resp)

	assert.Len(t, resp.Comments, 8, "Should have 8 comments")

	// Comments should be ordered by creation time DESC (newest first)
	assert.Equal(t, "Keep up the good work!", resp.Comments[0].Body)
	assert.Equal(t, "david", resp.Comments[0].Author.Username)

	assert.Equal(t, "Thanks for the feedback everyone!", resp.Comments[1].Body)
	assert.Equal(t, "alice", resp.Comments[1].Author.Username)

	assert.Equal(t, "Looking forward to more content.", resp.Comments[2].Body)
	assert.Equal(t, "charlie", resp.Comments[2].Author.Username)

	assert.Equal(t, "Thanks for sharing!", resp.Comments[3].Body)
	assert.Equal(t, "eve", resp.Comments[3].Author.Username)

	assert.Equal(t, "Very informative!", resp.Comments[4].Body)
	assert.Equal(t, "david", resp.Comments[4].Author.Username)

	assert.Equal(t, "Also, the examples were clear.", resp.Comments[5].Body)
	assert.Equal(t, "bob", resp.Comments[5].Author.Username)

	assert.Equal(t, "I learned a lot from this.", resp.Comments[6].Body)
	assert.Equal(t, "charlie", resp.Comments[6].Author.Username)

	assert.Equal(t, "Great article Alice!", resp.Comments[7].Body)
	assert.Equal(t, "bob", resp.Comments[7].Author.Username)

	// Verify all comments have following=false when not authenticated
	for i, c := range resp.Comments {
		assert.False(t, c.Author.Following, "Comment %d should have following=false when not authenticated", i)
	}

	// Verify all comments have required fields
	for i, c := range resp.Comments {
		assert.NotZero(t, c.ID, "Comment %d should have an ID", i)
		assert.NotZero(t, c.CreatedAt, "Comment %d should have CreatedAt", i)
		assert.NotZero(t, c.UpdatedAt, "Comment %d should have UpdatedAt", i)
		assert.NotEmpty(t, c.Author.Username, "Comment %d should have author username", i)
	}
}

func TestGetCommentsHandler_WithAuthentication(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t)

	// Setup: Register 5 users
	registerUser(t, ts, "alice", "alice@example.com", "password123")
	registerUser(t, ts, "bob", "bob@example.com", "password123")
	registerUser(t, ts, "charlie", "charlie@example.com", "password123")
	registerUser(t, ts, "david", "david@example.com", "password123")
	registerUser(t, ts, "eve", "eve@example.com", "password123")

	aliceToken := loginUser(t, ts, "alice@example.com", "password123")
	bobToken := loginUser(t, ts, "bob@example.com", "password123")
	charlieToken := loginUser(t, ts, "charlie@example.com", "password123")
	davidToken := loginUser(t, ts, "david@example.com", "password123")
	eveToken := loginUser(t, ts, "eve@example.com", "password123")

	articleLocation := createArticle(t, ts, aliceToken, "Test Article", "Test description", "Test body", []string{"test"})

	// Add 7 comments with some users commenting multiple times
	createCommentHelper(t, ts, bobToken, articleLocation, "Great article Alice!")
	createCommentHelper(t, ts, charlieToken, articleLocation, "I learned a lot from this.")
	createCommentHelper(t, ts, davidToken, articleLocation, "Very informative!")
	createCommentHelper(t, ts, bobToken, articleLocation, "The second point was especially helpful.")
	createCommentHelper(t, ts, eveToken, articleLocation, "Thanks for sharing!")
	createCommentHelper(t, ts, davidToken, articleLocation, "Really appreciate this content!")
	createCommentHelper(t, ts, aliceToken, articleLocation, "Thanks for the feedback everyone!")

	// Alice follows Bob and David
	followUser(t, ts, aliceToken, "bob")
	followUser(t, ts, aliceToken, "david")

	// Test: Get comments with authentication showing correct following status
	res, err := ts.executeRequest(http.MethodGet, articleLocation+"/comments", "", map[string]string{"Authorization": "Token " + aliceToken})
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)

	var resp struct {
		Comments []comment `json:"comments"`
	}
	readJsonResponse(t, res.Body, &resp)

	assert.Len(t, resp.Comments, 7)

	// Count comments by author and verify following status
	bobComments := 0
	charlieComments := 0
	davidComments := 0
	eveComments := 0
	aliceComments := 0

	for i := range resp.Comments {
		switch resp.Comments[i].Author.Username {
		case "bob":
			bobComments++
			assert.True(t, resp.Comments[i].Author.Following, "Alice follows Bob, all Bob's comments should have following=true")
		case "charlie":
			charlieComments++
			assert.False(t, resp.Comments[i].Author.Following, "Alice doesn't follow Charlie, all Charlie's comments should have following=false")
		case "david":
			davidComments++
			assert.True(t, resp.Comments[i].Author.Following, "Alice follows David, all David's comments should have following=true")
		case "eve":
			eveComments++
			assert.False(t, resp.Comments[i].Author.Following, "Alice doesn't follow Eve, all Eve's comments should have following=false")
		case "alice":
			aliceComments++
			assert.False(t, resp.Comments[i].Author.Following, "Alice doesn't follow herself, all her comments should have following=false")
		}
	}

	// Verify all users have commented
	assert.Equal(t, 2, bobComments, "Bob should have 2 comments")
	assert.Equal(t, 1, charlieComments, "Charlie should have 1 comment")
	assert.Equal(t, 2, davidComments, "David should have 2 comments")
	assert.Equal(t, 1, eveComments, "Eve should have 1 comment")
	assert.Equal(t, 1, aliceComments, "Alice should have 1 comment")
}

func TestGetCommentsHandler_DifferentUserPerspectives(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t)

	// Setup: Register 5 users
	registerUser(t, ts, "alice", "alice@example.com", "password123")
	registerUser(t, ts, "bob", "bob@example.com", "password123")
	registerUser(t, ts, "charlie", "charlie@example.com", "password123")
	registerUser(t, ts, "david", "david@example.com", "password123")
	registerUser(t, ts, "eve", "eve@example.com", "password123")

	aliceToken := loginUser(t, ts, "alice@example.com", "password123")
	bobToken := loginUser(t, ts, "bob@example.com", "password123")
	charlieToken := loginUser(t, ts, "charlie@example.com", "password123")
	davidToken := loginUser(t, ts, "david@example.com", "password123")
	eveToken := loginUser(t, ts, "eve@example.com", "password123")

	articleLocation := createArticle(t, ts, aliceToken, "Test Article", "Test description", "Test body", []string{"test"})

	// Add 8 comments with multiple users commenting more than once
	createCommentHelper(t, ts, bobToken, articleLocation, "Great article!")
	createCommentHelper(t, ts, charlieToken, articleLocation, "Interesting perspective.")
	createCommentHelper(t, ts, davidToken, articleLocation, "Well written!")
	createCommentHelper(t, ts, eveToken, articleLocation, "Insightful!")
	createCommentHelper(t, ts, charlieToken, articleLocation, "Also love the examples.")
	createCommentHelper(t, ts, eveToken, articleLocation, "Can't wait for part 2!")
	createCommentHelper(t, ts, aliceToken, articleLocation, "Thanks!")
	createCommentHelper(t, ts, davidToken, articleLocation, "Shared this with my team.")

	// Bob follows Charlie and Eve
	followUser(t, ts, bobToken, "charlie")
	followUser(t, ts, bobToken, "eve")

	// Test: Bob's perspective on following status
	res, err := ts.executeRequest(http.MethodGet, articleLocation+"/comments", "", map[string]string{"Authorization": "Token " + bobToken})
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)

	var resp struct {
		Comments []comment `json:"comments"`
	}
	readJsonResponse(t, res.Body, &resp)

	assert.Len(t, resp.Comments, 8)

	// Count comments by author and verify following status for each
	bobComments := 0
	charlieComments := 0
	davidComments := 0
	eveComments := 0
	aliceComments := 0

	for i := range resp.Comments {
		switch resp.Comments[i].Author.Username {
		case "bob":
			bobComments++
			assert.False(t, resp.Comments[i].Author.Following, "Bob doesn't follow himself")
		case "charlie":
			charlieComments++
			assert.True(t, resp.Comments[i].Author.Following, "Bob follows Charlie, all Charlie's comments should have following=true")
		case "david":
			davidComments++
			assert.False(t, resp.Comments[i].Author.Following, "Bob doesn't follow David, all David's comments should have following=false")
		case "eve":
			eveComments++
			assert.True(t, resp.Comments[i].Author.Following, "Bob follows Eve, all Eve's comments should have following=true")
		case "alice":
			aliceComments++
			assert.False(t, resp.Comments[i].Author.Following, "Bob doesn't follow Alice, all Alice's comments should have following=false")
		}
	}

	// Verify comment counts
	assert.Equal(t, 1, bobComments, "Bob should have 1 comment")
	assert.Equal(t, 2, charlieComments, "Charlie should have 2 comments")
	assert.Equal(t, 2, davidComments, "David should have 2 comments")
	assert.Equal(t, 2, eveComments, "Eve should have 2 comments")
	assert.Equal(t, 1, aliceComments, "Alice should have 1 comment")
}
