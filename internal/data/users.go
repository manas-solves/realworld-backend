package data

import (
	"context"
	"errors"
	"time"

	"github.com/manas-solves/realworld-backend/internal/validator"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrDuplicateEmail    = errors.New("duplicate email")
	ErrDuplicateUsername = errors.New("duplicate username")
)

var AnonymousUser = &User{}

type User struct {
	ID       int64    `json:"-"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Password password `json:"-"`
	Image    string   `json:"image"`
	Bio      string   `json:"bio"`
	Token    string   `json:"token"`
	Version  int      `json:"-"`
}

// Profile represents a user's public profile with follow status.
type Profile struct {
	Username  string `json:"username"`
	Bio       string `json:"bio"`
	Image     string `json:"image"`
	Following bool   `json:"following"`
}

// IsAnonymous returns true if the user is the special AnonymousUser user.
func (u *User) IsAnonymous() bool {
	return u == AnonymousUser
}

// ToProfile converts a User to a Profile with the specified following status.
func (u *User) ToProfile(following bool) Profile {
	return Profile{
		Username:  u.Username,
		Bio:       u.Bio,
		Image:     u.Image,
		Following: following,
	}
}

type password struct {
	plaintext *string
	hash      []byte
}

func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		return err
	}

	p.plaintext = &plaintextPassword
	p.hash = hash

	return nil
}

// Matches compares the plaintext password against the hash and returns true if they match.
func (p *password) Matches(plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}

	return true, nil
}

func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email != "", "email must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email must be a valid email address")
}

func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	v.Check(password != "", "password must be provided")
	v.Check(len(password) >= 8, "password must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password must not be more than 72 bytes long")
}

// ValidateUser checks the values provided by the user are valid. It performs validation on the
// Name, Email and Password fields.
func ValidateUser(v *validator.Validator, user User) {
	v.Check(user.Username != "", "username must be provided")
	v.Check(len(user.Username) <= 500, "name must not be more than 500 bytes long")

	ValidateEmail(v, user.Email)

	if user.Password.plaintext != nil {
		ValidatePasswordPlaintext(v, *user.Password.plaintext)
	}

	// If the password hash is ever nil, this will be due to a logic error in our codebase.
	if user.Password.hash == nil {
		panic("missing password hash for user")
	}
}

type UserStore struct {
	db        *pgxpool.Pool
	timeout   time.Duration
	userCache *UserCache
}

// Insert adds a new record in the users table.
func (s UserStore) Insert(user *User) error {
	query := `
		INSERT INTO users (username, email, password_hash, image, bio) 
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	args := []any{user.Username, user.Email, user.Password.hash, user.Image, user.Bio}

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	err := s.db.QueryRow(ctx, query, args...).Scan(&user.ID)
	if err != nil {
		switch {
		case err.Error() == `ERROR: duplicate key value violates unique constraint "users_email_key" (SQLSTATE 23505)`:
			return ErrDuplicateEmail
		case err.Error() == `ERROR: duplicate key value violates unique constraint "users_username_key" (SQLSTATE 23505)`:
			return ErrDuplicateUsername
		default:
			return err
		}
	}
	return nil
}

// GetByEmail retrieves a user by their email address.
func (s UserStore) GetByEmail(email string) (*User, error) {
	query := `
		SELECT id, username, email, password_hash, image, bio, version
		FROM users
		WHERE email = $1`

	var user User

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	err := s.db.QueryRow(ctx, query, email).Scan(&user.ID, &user.Username, &user.Email, &user.Password.hash, &user.Image, &user.Bio, &user.Version)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

// GetByID retrieves a user by their ID from the database.
// Uses cache if available, otherwise queries the database and caches the result.
func (s UserStore) GetByID(id int64) (*User, error) {
	// Try to get from cache first if cache is available
	if s.userCache != nil {
		if user, found := s.userCache.Get(id); found {
			return user, nil
		}
	}

	query := `
		SELECT id, username, email, password_hash, image, bio, version
		FROM users
		WHERE id = $1`

	var user User

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	err := s.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password.hash,
		&user.Image,
		&user.Bio,
		&user.Version,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	// Cache the user if cache is available
	if s.userCache != nil {
		s.userCache.Set(id, &user)
	}

	return &user, nil
}

// GetByUsername retrieves a user by their username from the database.
func (s UserStore) GetByUsername(username string) (*User, error) {
	query := `SELECT id, username, email, image, bio, version FROM users WHERE username = $1`
	var user User

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	err := s.db.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Image,
		&user.Bio,
		&user.Version,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}
	return &user, nil
}

// FollowUser creates a follow relationship between two users.
func (s UserStore) FollowUser(followerID, followedID int64) error {
	if followerID == followedID {
		return errors.New("cannot follow yourself")
	}
	query := `INSERT INTO follows (follower_id, followed_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()
	_, err := s.db.Exec(ctx, query, followerID, followedID)
	return err
}

// UnfollowUser removes a follow relationship between two users.
func (s UserStore) UnfollowUser(followerID, followedID int64) error {
	query := `DELETE FROM follows WHERE follower_id = $1 AND followed_id = $2`
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()
	_, err := s.db.Exec(ctx, query, followerID, followedID)
	return err
}

// IsFollowing checks if followerID is following followedID.
func (s UserStore) IsFollowing(followerID, followedID int64) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM follows WHERE follower_id = $1 AND followed_id = $2)`
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()
	var exists bool
	err := s.db.QueryRow(ctx, query, followerID, followedID).Scan(&exists)
	return exists, err
}

// Update updates an existing user record in the database.
// Invalidates the cache for the updated user.
func (s UserStore) Update(user *User) error {
	query := `
		UPDATE users
		SET username = $1, email = $2, password_hash = $3, image = $4, bio = $5, version = version + 1
		WHERE id = $6
		RETURNING version`
	args := []any{user.Username, user.Email, user.Password.hash, user.Image, user.Bio, user.ID}
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	err := s.db.QueryRow(ctx, query, args...).Scan(&user.Version)
	if err != nil {
		return err
	}

	// Invalidate cache after successful update
	if s.userCache != nil {
		s.userCache.Delete(user.ID)
	}

	return nil
}
