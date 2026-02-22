package main

import (
	"net/http"
	"testing"

	"github.com/manas-solves/realworld-backend/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type userResponse struct {
	User user `json:"user"`
}

type user struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Image    string `json:"image"`
	Bio      string `json:"bio"`
	Token    string `json:"token"`
}

type profile struct {
	Username  string `json:"username"`
	Bio       string `json:"bio"`
	Image     string `json:"image"`
	Following bool   `json:"following"`
}

type profileResponse struct {
	Profile profile `json:"profile"`
}

var seedUserRequest = `{
		"user": {
			"username": "Alice",
			"email": "alice@gmail.com",
			"password": "pa55word1234"
			}
		}`

func registerUser(t *testing.T, ts *testServer, username, email, password string) {
	t.Helper()
	register := `{"user":{"username":"` + username + `","email":"` + email + `","password":"` + password + `"}}`
	resp, err := ts.executeRequest(http.MethodPost, "/users", register, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
}

func loginUser(t *testing.T, ts *testServer, email, password string) string {
	t.Helper()
	login := `{"user":{"email":"` + email + `","password":"` + password + `"}}`
	resp, err := ts.executeRequest(http.MethodPost, "/users/login", login, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var userResp userResponse
	readJsonResponse(t, resp.Body, &userResp)
	return userResp.User.Token
}

func TestRegisterUserHandler(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)

	// Insert a seed user
	res, err := ts.executeRequest(http.MethodPost,
		"/users", seedUserRequest, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, res.StatusCode)

	testCases := []struct {
		name                   string
		jwtMaker               *dummyJWTMaker
		requestBody            string
		wantResponseStatusCode int
		wantResponse           any
	}{
		{
			name:                   "Valid request",
			requestBody:            `{"user":{"username":"Bob", "email":"bob@gmail.com", "password":"pa55word1234"}}`,
			jwtMaker:               &dummyJWTMaker{},
			wantResponseStatusCode: http.StatusCreated,
			wantResponse: userResponse{
				User: user{
					Username: "Bob",
					Email:    "bob@gmail.com",
					Image:    "",
					Bio:      "",
					Token:    "dummy-token",
				},
			},
		},
		{
			name:                   "Invalid request body",
			requestBody:            `{"name":"Alice", "email":"alice@gmail.com", "password":"pa55word1234"}`,
			wantResponseStatusCode: http.StatusBadRequest,
			wantResponse: errorResponse{
				Errors: []string{"body contains unknown key \"name\""},
			},
		},
		{
			name:                   "Invalid email",
			requestBody:            `{"user":{"username":"Bob", "email":"bob.gmail.com", "password":"pa55word1234"}}`,
			wantResponseStatusCode: http.StatusUnprocessableEntity,
			wantResponse: errorResponse{
				Errors: []string{"email must be a valid email address"},
			},
		},
		{
			name:                   "Invalid password with empty username",
			requestBody:            `{"user":{"username":"", "email":"abc@gmail.com", "password":"123"}}`,
			wantResponseStatusCode: http.StatusUnprocessableEntity,
			wantResponse: errorResponse{
				Errors: []string{"username must be provided", "password must be at least 8 bytes long"},
			},
		},
		{
			name:                   "Duplicate email",
			requestBody:            `{"user":{"username":"alice_new", "email":"alice@gmail.com", "password":"pa55word1234"}}`,
			wantResponseStatusCode: http.StatusUnprocessableEntity,
			wantResponse: errorResponse{
				Errors: []string{"a user with this email address already exists"},
			},
		},
		{
			name:                   "Duplicate username",
			requestBody:            `{"user":{"username":"Alice", "email":"alice_new@gmail.com", "password":"pa55word1234"}}`,
			wantResponseStatusCode: http.StatusUnprocessableEntity,
			wantResponse: errorResponse{
				Errors: []string{"a user with this username already exists"},
			},
		},
		{
			name:                   "Badly formed request body with unclosed JSON",
			requestBody:            `{"username":"Bob", "email":`,
			wantResponseStatusCode: http.StatusBadRequest,
			wantResponse: errorResponse{
				Errors: []string{"body contains badly-formed JSON"},
			},
		},
		{
			name:                   "Badly formed request body",
			requestBody:            `{"user": {"username":"Bob", "email"}`,
			wantResponseStatusCode: http.StatusBadRequest,
			wantResponse: errorResponse{
				Errors: []string{"body contains badly-formed JSON (at character 36)"},
			},
		},
		{
			name:                   "Token creation error",
			jwtMaker:               &dummyJWTMaker{CreateTokenErr: auth.ErrInvalidToken},
			requestBody:            `{"user":{"username":"Bob2", "email":"bob2@gmail.com", "password":"pa55word1234"}}`,
			wantResponseStatusCode: http.StatusInternalServerError,
			wantResponse: errorResponse{
				Errors: []string{"the server encountered a problem and could not process your request"},
			},
		},
	}

	for _, tc := range testCases {
		if tc.jwtMaker != nil {
			ts.app.jwtMaker = tc.jwtMaker
		}
		testHandler(t, ts, handlerTestcase{
			name:                   tc.name,
			requestUrlPath:         "/users",
			requestMethodType:      http.MethodPost,
			requestBody:            tc.requestBody,
			wantResponseStatusCode: tc.wantResponseStatusCode,
			wantResponse:           tc.wantResponse,
		})
	}
}

func TestLoginUserHandler(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)

	// Insert a seed user
	res, err := ts.executeRequest(http.MethodPost,
		"/users", seedUserRequest, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, res.StatusCode)

	testCases := []struct {
		name                   string
		jwtMaker               *dummyJWTMaker
		requestBody            string
		wantResponseStatusCode int
		wantResponse           any
	}{
		{
			name:                   "Valid request",
			requestBody:            `{"user":{"email":"alice@gmail.com", "password":"pa55word1234"}}`,
			jwtMaker:               &dummyJWTMaker{},
			wantResponseStatusCode: http.StatusOK,
			wantResponse: userResponse{
				User: user{
					Username: "Alice",
					Email:    "alice@gmail.com",
					Token:    "dummy-token",
					Image:    "",
					Bio:      "",
				},
			},
		},
		{
			name:                   "Email does not exist",
			requestBody:            `{"user":{"email":"alic@gmail.com", "password":"pa55word1234"}}`,
			wantResponseStatusCode: http.StatusUnauthorized,
			wantResponse: errorResponse{
				Errors: []string{"invalid authentication credentials"},
			},
		},
		{
			name:                   "Invalid password",
			requestBody:            `{"user":{"email":"alice@gmail.com", "password":"wrongpassword"}}`,
			wantResponseStatusCode: http.StatusUnauthorized,
			wantResponse: errorResponse{
				Errors: []string{"invalid authentication credentials"},
			},
		},
		{
			name:                   "Empty email and password",
			requestBody:            `{"user":{"email":"", "password":""}}`,
			wantResponseStatusCode: http.StatusUnprocessableEntity,
			wantResponse: errorResponse{
				Errors: []string{"email must be provided",
					"email must be a valid email address",
					"password must be provided",
					"password must be at least 8 bytes long",
				},
			},
		},
		{
			name:                   "Invalid request body",
			requestBody:            `{"user":{"name":"Alice", "password":"pa55word1234"}}`,
			wantResponseStatusCode: http.StatusBadRequest,
			wantResponse: errorResponse{
				Errors: []string{"body contains unknown key \"name\""},
			},
		},
		{
			name:                   "Token creation error",
			jwtMaker:               &dummyJWTMaker{CreateTokenErr: auth.ErrInvalidToken},
			requestBody:            `{"user":{"email":"alice@gmail.com", "password":"pa55word1234"}}`,
			wantResponseStatusCode: http.StatusInternalServerError,
			wantResponse: errorResponse{
				Errors: []string{"the server encountered a problem and could not process your request"},
			},
		},
	}

	for _, tc := range testCases {
		if tc.jwtMaker != nil {
			ts.app.jwtMaker = tc.jwtMaker
		}
		testHandler(t, ts, handlerTestcase{
			name:                   tc.name,
			requestUrlPath:         "/users/login",
			requestMethodType:      http.MethodPost,
			requestBody:            tc.requestBody,
			wantResponseStatusCode: tc.wantResponseStatusCode,
			wantResponse:           tc.wantResponse,
		})
	}
}

func TestGetCurrentUserHandler(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)
	// Register user Bob
	registerUser(t, ts, "Bob", "bob@example.com", "passwordbob")

	// Register user Alice
	registerUser(t, ts, "Alice", "alice@example.com", "passwordalice")

	// Login as Bob
	loginBob := `{"user":{"email":"bob@example.com","password":"passwordbob"}}`
	resp, err := ts.executeRequest(http.MethodPost, "/users/login", loginBob, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var loginRespBob userResponse
	readJsonResponse(t, resp.Body, &loginRespBob)
	tokenBob := loginRespBob.User.Token

	// Login as Alice
	loginAlice := `{"user":{"email":"alice@example.com","password":"passwordalice"}}`
	resp, err = ts.executeRequest(http.MethodPost, "/users/login", loginAlice, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var loginRespAlice userResponse
	readJsonResponse(t, resp.Body, &loginRespAlice)
	tokenAlice := loginRespAlice.User.Token

	testCases := []handlerTestcase{
		{
			name:                   "authenticated user Bob",
			requestUrlPath:         "/user",
			requestMethodType:      http.MethodGet,
			requestHeader:          map[string]string{"Authorization": "Token " + tokenBob},
			wantResponseStatusCode: http.StatusOK,
			wantResponse: userResponse{
				User: user{
					Username: "Bob",
					Email:    "bob@example.com",
					Token:    tokenBob,
				},
			},
		},
		{
			name:                   "authenticated user Alice",
			requestUrlPath:         "/user",
			requestMethodType:      http.MethodGet,
			requestHeader:          map[string]string{"Authorization": "Token " + tokenAlice},
			wantResponseStatusCode: http.StatusOK,
			wantResponse: userResponse{
				User: user{
					Username: "Alice",
					Email:    "alice@example.com",
					Token:    tokenAlice,
				},
			},
		},
		{
			name:                   "anonymous user",
			requestUrlPath:         "/user",
			requestMethodType:      http.MethodGet,
			wantResponseStatusCode: http.StatusUnauthorized,
			wantResponse: errorResponse{
				Errors: []string{"invalid or missing authentication token"},
			},
		},
	}

	testHandler(t, ts, testCases...)
}

func TestGetProfileUserHandler(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)

	// Register Alice, Bob and Charlie
	registerUser(t, ts, "Alice", "alice@example.com", "alicepassword")
	registerUser(t, ts, "Bob", "bob@example.com", "bobpassword")
	registerUser(t, ts, "Charlie", "charlie@example.com", "charliepassword")

	// Login as Bob
	bobToken := loginUser(t, ts, "bob@example.com", "bobpassword")

	// Make Bob follow Alice
	headers := map[string]string{"Authorization": "Token " + bobToken}
	resp, err := ts.executeRequest(http.MethodPost, "/profiles/Alice/follow", "", headers)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	testCases := []handlerTestcase{
		{
			name:                   "anonymous user gets Alice's profile",
			requestUrlPath:         "/profiles/Alice",
			requestMethodType:      http.MethodGet,
			wantResponseStatusCode: http.StatusOK,
			wantResponse: profileResponse{
				Profile: profile{
					Username:  "Alice",
					Bio:       "",
					Image:     "",
					Following: false,
				},
			},
		},
		{
			name:                   "authenticated user follows Alice and gets profile",
			requestUrlPath:         "/profiles/Alice",
			requestMethodType:      http.MethodGet,
			requestHeader:          map[string]string{"Authorization": "Token " + bobToken},
			wantResponseStatusCode: http.StatusOK,
			wantResponse: profileResponse{
				Profile: profile{
					Username:  "Alice",
					Bio:       "",
					Image:     "",
					Following: true,
				},
			},
		},
		{
			name:                   "authenticated user gets charlie's profile and not following",
			requestUrlPath:         "/profiles/Charlie",
			requestMethodType:      http.MethodGet,
			requestHeader:          map[string]string{"Authorization": "Token " + bobToken},
			wantResponseStatusCode: http.StatusOK,
			wantResponse: profileResponse{
				Profile: profile{
					Username:  "Charlie",
					Bio:       "",
					Image:     "",
					Following: false,
				},
			},
		},
		{
			name:                   "profile for non-existent user returns 404",
			requestUrlPath:         "/profiles/nonexistent",
			requestMethodType:      http.MethodGet,
			wantResponseStatusCode: http.StatusNotFound,
		},
	}

	testHandler(t, ts, testCases...)
}

func TestFollowUserHandler(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)

	// Register Alice and Bob
	registerUser(t, ts, "Alice", "alice@example.com", "alicepassword")
	registerUser(t, ts, "Bob", "bob@example.com", "bobpassword")

	// Login as Bob
	bobToken := loginUser(t, ts, "bob@example.com", "bobpassword")

	testCases := []handlerTestcase{
		{
			name:                   "authenticated user follows Alice",
			requestUrlPath:         "/profiles/Alice/follow",
			requestMethodType:      http.MethodPost,
			requestHeader:          map[string]string{"Authorization": "Token " + bobToken},
			wantResponseStatusCode: http.StatusOK,
			wantResponse: profileResponse{
				Profile: profile{
					Username:  "Alice",
					Bio:       "",
					Image:     "",
					Following: true,
				},
			},
		},
		{
			name:                   "anonymous user cannot follow Alice",
			requestUrlPath:         "/profiles/Alice/follow",
			requestMethodType:      http.MethodPost,
			wantResponseStatusCode: http.StatusUnauthorized,
			wantResponse: errorResponse{
				Errors: []string{"invalid or missing authentication token"},
			},
		},
		{
			name:                   "authenticated user cannot follow non-existent user",
			requestUrlPath:         "/profiles/NonExistent/follow",
			requestMethodType:      http.MethodPost,
			requestHeader:          map[string]string{"Authorization": "Token " + bobToken},
			wantResponseStatusCode: http.StatusNotFound,
		},
		{
			name:                   "user cannot follow themselves",
			requestUrlPath:         "/profiles/Bob/follow",
			requestMethodType:      http.MethodPost,
			requestHeader:          map[string]string{"Authorization": "Token " + bobToken},
			wantResponseStatusCode: http.StatusUnprocessableEntity,
			wantResponse: errorResponse{
				Errors: []string{"cannot follow yourself"},
			},
		},
	}
	testHandler(t, ts, testCases...)
}

func TestUnfollowUserHandler(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)

	// Register Alice and Bob
	registerUser(t, ts, "Alice", "alice@example.com", "alicepassword")
	registerUser(t, ts, "Bob", "bob@example.com", "bobpassword")

	// Login as Bob
	loginBob := `{"user":{"email":"bob@example.com","password":"bobpassword"}}`
	resp, err := ts.executeRequest(http.MethodPost, "/users/login", loginBob, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var loginResp userResponse
	readJsonResponse(t, resp.Body, &loginResp)
	bobToken := loginResp.User.Token

	// Bob follows Alice
	headers := map[string]string{"Authorization": "Token " + bobToken}
	resp, err = ts.executeRequest(http.MethodPost, "/profiles/Alice/follow", "", headers)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var followResp profileResponse
	readJsonResponse(t, resp.Body, &followResp)
	require.Equal(t, true, followResp.Profile.Following)

	testCases := []handlerTestcase{
		{
			name:                   "authenticated user unfollows Alice",
			requestUrlPath:         "/profiles/Alice/follow",
			requestMethodType:      http.MethodDelete,
			requestHeader:          map[string]string{"Authorization": "Token " + bobToken},
			wantResponseStatusCode: http.StatusOK,
			wantResponse: profileResponse{
				Profile: profile{
					Username:  "Alice",
					Bio:       "",
					Image:     "",
					Following: false,
				},
			},
		},
		{
			name:                   "anonymous user cannot unfollow Alice",
			requestUrlPath:         "/profiles/Alice/follow",
			requestMethodType:      http.MethodDelete,
			wantResponseStatusCode: http.StatusUnauthorized,
			wantResponse: errorResponse{
				Errors: []string{"invalid or missing authentication token"},
			},
		},
		{
			name:                   "authenticated user cannot unfollow non-existent user",
			requestUrlPath:         "/profiles/NonExistent/follow",
			requestMethodType:      http.MethodDelete,
			requestHeader:          map[string]string{"Authorization": "Token " + bobToken},
			wantResponseStatusCode: http.StatusNotFound,
		},
		{
			name:                   "user cannot unfollow themselves",
			requestUrlPath:         "/profiles/Bob/follow",
			requestMethodType:      http.MethodDelete,
			requestHeader:          map[string]string{"Authorization": "Token " + bobToken},
			wantResponseStatusCode: http.StatusOK, // Unfollowing yourself is a no-op, returns not following
			wantResponse: profileResponse{
				Profile: profile{
					Username:  "Bob",
					Bio:       "",
					Image:     "",
					Following: false,
				},
			},
		},
	}
	testHandler(t, ts, testCases...)
}

func TestUpdateUserHandler(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)

	registerUser(t, ts, "Alice", "alice@example.com", "alicepassword")
	registerUser(t, ts, "Bob", "bob@example.com", "bobpassword")
	registerUser(t, ts, "Charlie", "charlie@example.com", "charliepassword")

	aliceToken := loginUser(t, ts, "alice@example.com", "alicepassword")
	bobToken := loginUser(t, ts, "bob@example.com", "bobpassword")
	charlieToken := loginUser(t, ts, "charlie@example.com", "charliepassword")

	testCases := []handlerTestcase{
		{
			name:                   "update username and bio",
			requestUrlPath:         "/user",
			requestMethodType:      http.MethodPut,
			requestHeader:          map[string]string{"Authorization": "Token " + aliceToken},
			requestBody:            `{"user":{"username":"AliceNew","bio":"new bio"}}`,
			wantResponseStatusCode: http.StatusOK,
			additionalChecks: func(t *testing.T, res *http.Response) {

				// deserialize the response body
				var userResp userResponse
				readJsonResponse(t, res.Body, &userResp)
				assert.Equal(t, "AliceNew", userResp.User.Username, "username should be updated")
				assert.Equal(t, "new bio", userResp.User.Bio, "bio should be updated")
				assert.NotEmpty(t, userResp.User.Token, "token should not be empty")
				assert.Equal(t, "alice@example.com", userResp.User.Email, "email should remain unchanged")
			},
		},
		{
			name:                   "update email and image",
			requestUrlPath:         "/user",
			requestMethodType:      http.MethodPut,
			requestHeader:          map[string]string{"Authorization": "Token " + bobToken},
			requestBody:            `{"user":{"email":"bob2@example.com","image":"https://img.com/a.png"}}`,
			wantResponseStatusCode: http.StatusOK,
			additionalChecks: func(t *testing.T, res *http.Response) {
				// deserialize the response body
				var userResp userResponse
				readJsonResponse(t, res.Body, &userResp)
				assert.Equal(t, "Bob", userResp.User.Username, "username should remain unchanged")
				assert.Equal(t, "bob2@example.com", userResp.User.Email, "email should be updated")
				assert.Equal(t, "https://img.com/a.png", userResp.User.Image, "image should be updated")
				assert.NotEmpty(t, userResp.User.Token, "token should not be empty")
				assert.Equal(t, "", userResp.User.Bio, "bio should remain unchanged")
			},
		},
		{
			name:                   "update password only",
			requestUrlPath:         "/user",
			requestMethodType:      http.MethodPut,
			requestHeader:          map[string]string{"Authorization": "Token " + charlieToken},
			requestBody:            `{"user":{"password":"newpassword123"}}`,
			wantResponseStatusCode: http.StatusOK,
			additionalChecks: func(t *testing.T, res *http.Response) {
				// deserialize the response body
				var userResp userResponse
				readJsonResponse(t, res.Body, &userResp)
				assert.Equal(t, "Charlie", userResp.User.Username, "username should remain unchanged")
				assert.Equal(t, "charlie@example.com", userResp.User.Email, "email should remain unchanged")
				assert.Equal(t, "", userResp.User.Image, "image should remain unchanged")
				assert.Equal(t, "", userResp.User.Bio, "bio should remain unchanged")
				assert.NotEmpty(t, userResp.User.Token, "token should not be empty")
			},
		},
		{
			name:                   "invalid email",
			requestUrlPath:         "/user",
			requestMethodType:      http.MethodPut,
			requestHeader:          map[string]string{"Authorization": "Token " + aliceToken},
			requestBody:            `{"user":{"email":"bademail"}}`,
			wantResponseStatusCode: http.StatusUnprocessableEntity,
			wantResponse: errorResponse{
				Errors: []string{"email must be a valid email address"},
			},
		},
		{
			name:                   "invalid password",
			requestUrlPath:         "/user",
			requestMethodType:      http.MethodPut,
			requestHeader:          map[string]string{"Authorization": "Token " + bobToken},
			requestBody:            `{"user":{"password":"short"}}`,
			wantResponseStatusCode: http.StatusUnprocessableEntity,
			wantResponse: errorResponse{
				Errors: []string{"password must be at least 8 bytes long"},
			},
		},
		{
			name:                   "unauthenticated user",
			requestUrlPath:         "/user",
			requestMethodType:      http.MethodPut,
			requestBody:            `{"user":{"bio":"should fail"}}`,
			wantResponseStatusCode: http.StatusUnauthorized,
			wantResponse: errorResponse{
				Errors: []string{"invalid or missing authentication token"},
			},
		},
	}
	testHandler(t, ts, testCases...)
}
