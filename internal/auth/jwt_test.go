package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJWTMaker(t *testing.T) {
	t.Run("Valid secret key", func(t *testing.T) {
		maker, err := NewJWTMaker("this-is-a-valid-secret-key-32-chars", "test-issuer")
		require.NoError(t, err)
		require.NotNil(t, maker)
	})

	t.Run("Secret key too short", func(t *testing.T) {
		maker, err := NewJWTMaker("short", "test-issuer")
		require.Error(t, err)
		require.Nil(t, maker)
		assert.Equal(t, ErrInvalidSecretKey, err)
	})
}

func TestJWTMaker_CreateToken(t *testing.T) {
	maker, err := NewJWTMaker("this-is-a-valid-secret-key-32-chars", "test-issuer")
	require.NoError(t, err)

	userID := int64(123)
	duration := 5 * time.Minute

	token, err := maker.CreateToken(userID, duration)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Verify the token
	claims, err := maker.VerifyToken(token)
	require.NoError(t, err)
	require.NotNil(t, claims)

	// Validate all claims
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, "123", claims.Subject)
	assert.Equal(t, "test-issuer", claims.Issuer)
	assert.Contains(t, claims.Audience, "test-issuer")
	assert.NotEmpty(t, claims.ID) // JTI should be set
	assert.True(t, claims.ExpiresAt.Time.After(time.Now()))
	assert.True(t, claims.IssuedAt.Time.Before(time.Now().Add(time.Second)))
}

func TestJWTMaker_VerifyToken(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func() (string, *JWTMaker)
		expectedErr error
	}{
		{
			name: "Valid token",
			setup: func() (string, *JWTMaker) {
				tm, _ := NewJWTMaker("this-is-a-valid-secret-key-32-chars", "test-issuer")
				token, _ := tm.CreateToken(1, 5*time.Minute)
				return token, tm
			},
			expectedErr: nil,
		},
		{
			name: "Expired token",
			setup: func() (string, *JWTMaker) {
				tm, _ := NewJWTMaker("this-is-a-valid-secret-key-32-chars", "test-issuer")
				token, _ := tm.CreateToken(1, -5*time.Minute)
				return token, tm
			},
			expectedErr: ErrExpiredToken,
		},
		{
			name: "Invalid secret key",
			setup: func() (string, *JWTMaker) {
				tm, _ := NewJWTMaker("this-is-a-valid-secret-key-32-chars", "test-issuer")
				token, _ := tm.CreateToken(1, 5*time.Minute)
				tm.secretKey = "different-secret-key-32-chars-lo"
				return token, tm
			},
			expectedErr: ErrInvalidToken,
		},
		{
			name: "Invalid issuer",
			setup: func() (string, *JWTMaker) {
				tm, _ := NewJWTMaker("this-is-a-valid-secret-key-32-chars", "test-issuer")
				token, _ := tm.CreateToken(1, 5*time.Minute)
				tm.issuer = "invalid-issuer"
				return token, tm
			},
			expectedErr: ErrInvalidToken,
		},
		{
			name: "Invalid audience",
			setup: func() (string, *JWTMaker) {
				tm, _ := NewJWTMaker("this-is-a-valid-secret-key-32-chars", "test-issuer")
				token, _ := tm.CreateToken(1, 5*time.Minute)
				tm.audience = "invalid-audience"
				return token, tm
			},
			expectedErr: ErrInvalidToken,
		},
		{
			name: "Malformed token",
			setup: func() (string, *JWTMaker) {
				tm, _ := NewJWTMaker("this-is-a-valid-secret-key-32-chars", "test-issuer")
				return "malformed.token", tm
			},
			expectedErr: ErrInvalidToken,
		},
		{
			name: "Empty token",
			setup: func() (string, *JWTMaker) {
				tm, _ := NewJWTMaker("this-is-a-valid-secret-key-32-chars", "test-issuer")
				return "", tm
			},
			expectedErr: ErrInvalidToken,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			token, tm := tc.setup()
			claims, err := tm.VerifyToken(token)
			if tc.expectedErr != nil {
				require.Error(t, err)
				require.Nil(t, claims)
				assert.Equal(t, tc.expectedErr, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, claims)
				assert.Equal(t, int64(1), claims.UserID)
				assert.Equal(t, "test-issuer", claims.Issuer)
			}
		})
	}
}
