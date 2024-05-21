package lib

import (
	"crypto/rand"
	"crypto/rsa"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Generate a JWT for a valid UUID
func TestCreateJWTWithValidUUID(t *testing.T) {
	validUUID := uuid.New()
	jwtString, err := CreateJWT(validUUID)
	assert.NoError(t, err)
	assert.NotEmpty(t, jwtString)
}

// Handle invalid UUID formats gracefully
func TestCreateJWTWithInvalidUUID(t *testing.T) {
	invalidUUID := uuid.Nil
	jwtString, err := CreateJWT(invalidUUID)
	assert.NoError(t, err)
	assert.NotEmpty(t, jwtString)
}

// Token is valid and correctly signed with the expected HMAC method
func TestValidTokenWithHMAC(t *testing.T) {
	userUUID := uuid.New()
	tokenString, err := CreateJWT(userUUID)
	if err != nil {
		t.Fatalf("Error creating JWT: %v", err)
	}

	resultUUID, err := CheckJWT(tokenString)
	if err != nil {
		t.Fatalf("Error checking JWT: %v", err)
	}

	if resultUUID != userUUID {
		t.Errorf("Expected UUID %v, got %v", userUUID, resultUUID)
	}
}

// Token uses an unexpected signing method, such as RSA instead of HMAC
func TestTokenWithUnexpectedSigningMethod(t *testing.T) {
	// Create a token with RSA signing method
	claims := JWTClaims{
		UserID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	rsaPrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	tokenString, err := token.SignedString(rsaPrivateKey)
	if err != nil {
		t.Fatalf("Error signing JWT with RSA: %v", err)
	}

	_, err = CheckJWT(tokenString)
	if err == nil || !strings.Contains(err.Error(), "unexpected signing method") {
		t.Errorf("Expected signing method error, got %v", err)
	}
}
