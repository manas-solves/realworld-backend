package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/manas-solves/realworld-backend/internal/auth"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

type errorResponse struct {
	Errors []string `json:"errors"`
}

type testServer struct {
	router http.Handler
	app    *application
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	// connect to the root db to create a new test db
	// generate a random db name of length 8
	dbName := "testdb_" + uuid.New().String()[:8]
	dsn := fmt.Sprintf("postgres://postgres:postgres@localhost:5432/%s?sslmode=disable", dbName)

	// create the database
	rootDB, err := pgxpool.New(context.Background(), "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	require.NoError(t, err)
	_, err = rootDB.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE %s;", dbName))
	require.NoError(t, err)
	t.Logf("created test database %s", dbName)

	// delete the database at the end of the test
	t.Cleanup(func() {
		rootDB.Exec(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE);", dbName))
		t.Logf("dropped test database %s", dbName)
		rootDB.Close()
	})

	// Run migrations on the test database using golang migrate
	t.Log("running migrations on test database...")
	db, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err)
	sqlDB := stdlib.OpenDBFromPool(db)
	driver, err := pgx.WithInstance(sqlDB, &pgx.Config{})
	require.NoError(t, err)
	m, err := migrate.NewWithDatabaseInstance("file://../../migrations", dbName, driver)
	require.NoError(t, err)
	err = m.Up()
	require.NoError(t, err)
	t.Log("migrations applied successfully")

	// close all connections
	m.Close()
	sqlDB.Close()
	db.Close()

	cfg := appConfig{
		env: "development",
		db: dbConfig{
			dsn:          dsn,
			maxIdleTime:  15 * time.Minute,
			maxOpenConns: 25,
			timeout:      30 * time.Second,
		},
		jwtMaker: jwtMakerConfig{
			secretKey:      "test-secret-key-must-be-32-chars-long",
			issuer:         "conduit_tests",
			accessDuration: 24 * time.Hour,
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := newApplication(cfg, logger)

	t.Logf("setting up test server...")
	return &testServer{
		router: app.routes(),
		app:    app,
	}
}

func (ts *testServer) executeRequest(method, urlPath, body string, requestHeader map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, urlPath, strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	// convert requestHeader map to http.Header
	header := http.Header{}
	for key, val := range requestHeader {
		header.Add(key, val)
	}
	req.Header = header

	rr := httptest.NewRecorder()
	ts.router.ServeHTTP(rr, req)
	return rr.Result(), nil
}

func readJsonResponse(t *testing.T, body io.Reader, dst any) {
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	err := dec.Decode(dst)
	require.NoError(t, err)
}

type dummyJWTMaker struct {
	TokenToReturn  string
	ClaimsToReturn *auth.Claims
	CreateTokenErr error
	VerifyTokenErr error
}

func (d *dummyJWTMaker) CreateToken(userID int64, duration time.Duration) (string, error) {
	if d.CreateTokenErr != nil {
		return "", d.CreateTokenErr
	}
	if d.TokenToReturn != "" {
		return d.TokenToReturn, nil
	}
	return "dummy-token", nil
}

func (d *dummyJWTMaker) VerifyToken(tokenString string) (*auth.Claims, error) {
	if d.VerifyTokenErr != nil {
		return nil, d.VerifyTokenErr
	}
	if d.ClaimsToReturn != nil {
		return d.ClaimsToReturn, nil
	}
	return &auth.Claims{UserID: 1}, nil
}

// createCommentHelper is a test helper that creates a comment on an article
func createCommentHelper(t *testing.T, ts *testServer, token, articleLocation, body string) {
	t.Helper()

	requestBody := `{"comment": {"body": "` + body + `"}}`
	headers := map[string]string{
		"Authorization": "Token " + token,
	}

	res, err := ts.executeRequest(http.MethodPost, articleLocation+"/comments", requestBody, headers)
	require.NoError(t, err)
	defer res.Body.Close() //nolint: errcheck

	require.Equal(t, http.StatusCreated, res.StatusCode)
}

// followUser is a test helper that makes one user follow another
func followUser(t *testing.T, ts *testServer, token, username string) {
	t.Helper()

	headers := map[string]string{
		"Authorization": "Token " + token,
	}

	res, err := ts.executeRequest(http.MethodPost, "/profiles/"+username+"/follow", "", headers)
	require.NoError(t, err)
	defer res.Body.Close() //nolint: errcheck

	require.Equal(t, http.StatusOK, res.StatusCode)
}

// favoriteArticleHelper is a test helper that favorites an article
func favoriteArticleHelper(t *testing.T, ts *testServer, token, slug string) {
	t.Helper()

	headers := map[string]string{
		"Authorization": "Token " + token,
	}

	res, err := ts.executeRequest(http.MethodPost, "/articles/"+slug+"/favorite", "", headers)
	require.NoError(t, err)
	defer res.Body.Close() //nolint: errcheck

	require.Equal(t, http.StatusOK, res.StatusCode)
}
