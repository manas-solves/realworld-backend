package data

import (
	"context"
	"time"

	"github.com/manas-solves/realworld-backend/internal/validator"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Comment struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	ArticleID int64     `json:"-"`
	AuthorID  int64     `json:"-"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Author    Profile   `json:"author"`
}

func ValidateComment(v *validator.Validator, comment *Comment) {
	v.Check(validator.NotEmptyOrWhitespace(comment.Body),
		"Body must not be empty or whitespace only")
}

type CommentStore struct {
	db      *pgxpool.Pool
	timeout time.Duration
}

// InsertAndReturn inserts a comment and populates it with database-generated fields and author details.
// Modifies the input comment object in place and uses currentUser from context instead of querying the database.
func (s *CommentStore) InsertAndReturn(comment *Comment, currentUser *User) (*Comment, error) {
	query := `
		INSERT INTO comments (body, article_id, author_id)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`

	args := []any{comment.Body, comment.ArticleID, comment.AuthorID}

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	// Scan only the fields we don't already have into the input object
	err := s.db.QueryRow(ctx, query, args...).Scan(&comment.ID, &comment.CreatedAt, &comment.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Use author information from currentUser context instead of querying database
	// Following is always false for newly created comments (user doesn't follow themselves)
	comment.Author = currentUser.ToProfile(false)

	return comment, nil
}

// GetByArticleID retrieves all comments for an article by its article ID.
// Returns comments with author details, ordered by creation time (newest first).
// Uses JOIN to efficiently fetch author information in a single query.
func (s *CommentStore) GetByArticleID(articleID int64) ([]Comment, error) {
	query := `
		SELECT c.id, c.body, c.article_id, c.author_id, c.created_at, c.updated_at,
		       u.username, u.bio, u.image
		FROM comments c
		JOIN users u ON c.author_id = u.id
		WHERE c.article_id = $1
		ORDER BY c.created_at DESC
	`

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	rows, err := s.db.Query(ctx, query, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var comment Comment
		var author Profile

		err := rows.Scan(
			&comment.ID,
			&comment.Body,
			&comment.ArticleID,
			&comment.AuthorID,
			&comment.CreatedAt,
			&comment.UpdatedAt,
			&author.Username,
			&author.Bio,
			&author.Image,
		)
		if err != nil {
			return nil, err
		}

		comment.Author = author
		comments = append(comments, comment)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no comments found
	if comments == nil {
		comments = []Comment{}
	}

	return comments, nil
}

// SetFollowingStatus efficiently checks and sets the following status for all comment authors.
// Uses a single query with IN clause to check all authors at once.
func (s *CommentStore) SetFollowingStatus(comments []Comment, currentUserID int64) error {
	if len(comments) == 0 || currentUserID == 0 {
		return nil
	}

	// Collect unique author IDs
	authorIDsMap := make(map[int64]bool)
	for _, comment := range comments {
		authorIDsMap[comment.AuthorID] = true
	}

	// Convert to slice for query
	authorIDs := make([]int64, 0, len(authorIDsMap))
	for id := range authorIDsMap {
		authorIDs = append(authorIDs, id)
	}

	// Bulk check following status for all authors
	query := `
		SELECT followed_id
		FROM follows
		WHERE followed_id = ANY($1) AND follower_id = $2
	`

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	rows, err := s.db.Query(ctx, query, authorIDs, currentUserID)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Build a set of author IDs that the current user is following
	followingSet := make(map[int64]bool)
	for rows.Next() {
		var authorID int64
		if err := rows.Scan(&authorID); err != nil {
			return err
		}
		followingSet[authorID] = true
	}

	if err = rows.Err(); err != nil {
		return err
	}

	// Update following status for each comment
	for i := range comments {
		comments[i].Author.Following = followingSet[comments[i].AuthorID]
	}

	return nil
}
