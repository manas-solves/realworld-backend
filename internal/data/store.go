package data

import (
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type ModelStore struct {
	Users    UserStoreInterface
	Articles ArticleStoreInterface
	Tags     TagStoreInterface
	Comments CommentStoreInterface
}

func NewModelStore(db *pgxpool.Pool, timeout time.Duration, userCache *UserCache) ModelStore {
	return ModelStore{
		Users:    &UserStore{db: db, timeout: timeout, userCache: userCache},
		Articles: &ArticleStore{db: db, timeout: timeout},
		Tags:     &TagStore{db: db, timeout: timeout},
		Comments: &CommentStore{db: db, timeout: timeout},
	}
}

type UserStoreInterface interface {
	// Insert a new record into the users table.
	Insert(user *User) error
	// GetByEmail returns a specific record from the users table.
	GetByEmail(email string) (*User, error)
	// GetByID retrieves a specific record from the users table by ID.
	GetByID(id int64) (*User, error)
	// GetByUsername retrieves a specific record from the users table by username.
	GetByUsername(username string) (*User, error)
	// FollowUser records that a user is following another user
	FollowUser(followerID, followedID int64) error
	// UnfollowUser records that a user has unfollowed another user
	UnfollowUser(followerID, followedID int64) error
	// IsFollowing checks if a user is following another user
	IsFollowing(followerID, followedID int64) (bool, error)
	// Update an existing user record.
	Update(user *User) error
}

type ArticleStoreInterface interface {
	// InsertAndReturn inserts an article and returns the complete article with author details in a single query.
	// This is more efficient than Insert followed by GetBySlug as it eliminates an extra database round trip.
	InsertAndReturn(article *Article, currentUser *User) (*Article, error)
	// GetIDBySlug retrieves just the article ID by its slug (lightweight alternative to GetBySlug).
	GetIDBySlug(slug string) (int64, error)
	// GetBySlug retrieves a specific record from the articles table by slug.
	GetBySlug(slug string, currentUser *User) (*Article, error)
	// List retrieves articles with optional filtering and pagination.
	List(filters ArticleFilters, currentUser *User) ([]Article, int, error)
	// FavoriteBySlug favorites the article with the given slug for the user and returns the updated article.
	FavoriteBySlug(slug string, userID int64) (*Article, error)
	// UnfavoriteBySlug unfavorites the article with the given slug for the user and returns the updated article.
	UnfavoriteBySlug(slug string, userID int64) (*Article, error)
	// DeleteBySlug deletes the article with the given slug.
	DeleteBySlug(slug string, userID int64) error
	// Update an existing article record.
	Update(article *Article) error
	// InsertTags inserts tags into the tags table (used for async operations).
	InsertTags(tags ...string) error
}

type TagStoreInterface interface {
	// GetAll retrieves all tags from the tags table.
	GetAll() ([]string, error)
}

type CommentStoreInterface interface {
	// InsertAndReturn inserts a comment and returns it with author details populated from currentUser.
	// Uses the currentUser from context instead of querying the database for author information.
	InsertAndReturn(comment *Comment, currentUser *User) (*Comment, error)
	// GetByArticleID retrieves all comments with author details for an article by its article ID.
	GetByArticleID(articleID int64) ([]Comment, error)
	// SetFollowingStatus efficiently checks and sets the following status for all comment authors.
	SetFollowingStatus(comments []Comment, currentUserID int64) error
}
