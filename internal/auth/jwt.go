package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken     = errors.New("token is invalid")
	ErrExpiredToken     = errors.New("token has expired")
	ErrInvalidSecretKey = errors.New("secret key must be at least 32 characters long")
)

type JWTMaker struct {
	secretKey     string
	issuer        string
	audience      string
	signingMethod jwt.SigningMethod
}

type Claims struct {
	UserID int64 `json:"uid"` // Custom claim for user ID
	jwt.RegisteredClaims
}

// NewJWTMaker creates a new JWTMaker with the given secret key and issuer.
// Returns an error if the secret key is less than 32 characters (256 bits for HMAC-SHA256).
func NewJWTMaker(secretKey string, issuer string) (*JWTMaker, error) {
	// HMAC-SHA256 should use at least 256 bits (32 bytes) for security
	if len(secretKey) < 32 {
		return nil, ErrInvalidSecretKey
	}

	return &JWTMaker{
		secretKey:     secretKey,
		issuer:        issuer,
		audience:      issuer, // Use issuer as audience by default
		signingMethod: jwt.SigningMethodHS256,
	}, nil
}

// CreateToken generates a new JWT access token for the given user ID and duration.
// It signs the token with the secret key and includes standard claims (iss, aud, sub, jti).
// It uses the HS256 signing method.
func (maker *JWTMaker) CreateToken(userID int64, duration time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),             // Standard way to identify the user
			Audience:  jwt.ClaimStrings{maker.audience},      // Who can use this token
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)), // Token expiration
			IssuedAt:  jwt.NewNumericDate(now),               // When token was issued
			NotBefore: jwt.NewNumericDate(now),               // Token not valid before this time
			Issuer:    maker.issuer,                          // Who issued the token
			ID:        uuid.New().String(),                   // Unique token ID (JTI) for tracking/revocation
		},
	}

	token := jwt.NewWithClaims(maker.signingMethod, claims)
	return token.SignedString([]byte(maker.secretKey))
}

// VerifyToken checks the validity of the given token string and returns the claims if valid.
// It validates the signing algorithm, issuer, and audience claims.
// It returns an error if the token is invalid or expired.
func (maker *JWTMaker) VerifyToken(tokenString string) (*Claims, error) {
	keyFunc := func(token *jwt.Token) (any, error) {
		// Prevent algorithm confusion attacks by validating the signing method
		if token.Method.Alg() != maker.signingMethod.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(maker.secretKey), nil
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, keyFunc)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Validate issuer
	if maker.issuer != "" && claims.Issuer != maker.issuer {
		return nil, ErrInvalidToken
	}

	// Validate audience (access tokens should have the standard audience)
	expectedAudience := maker.audience
	validAudience := false
	for _, aud := range claims.Audience {
		if aud == expectedAudience {
			validAudience = true
			break
		}
	}
	if !validAudience {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
