package lib

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var secret = "PaBjK!7K$&qMUMTb"

type JWTClaims struct {
	jwt.RegisteredClaims
	UserID uuid.UUID `json:"user_id"`
}

// CreateJWT generates a JSON Web Token (JWT) for the given userUUID.
//
// Parameters:
// - userUUID: The UUID of the user for whom the JWT is being generated.
//
// Returns:
// - string: The generated JWT.
// - error: An error if there was a problem generating the JWT.
func CreateJWT(userUUID uuid.UUID) (string, error) {
	claims := JWTClaims{
		UserID: userUUID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return jwtString, nil
}

// CheckJWT verifies the validity of a JSON Web Token (JWT) and extracts the UUID of the user.
//
// Parameters:
// - tokenString: The string representation of the JWT to be checked.
//
// Returns:
// - uuid.UUID: The UUID of the user contained in the JWT.
// - error: An error if there was a problem parsing or validating the JWT.
func CheckJWT(tokenString string) (uuid.UUID, error) {
	claims := JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return uuid.Nil, err
	}

	if !token.Valid {
		return uuid.Nil, err
	}

	return claims.UserID, nil
}

// IsTableExist checks if a table exists in the database.
//
// Parameters:
// - p: A pointer to a pgxpool.Pool object representing the database connection pool.
// - table: A string representing the name of the table to check.
//
// Returns:
// - bool: True if the table exists, false otherwise.
func IsTableExist(p *pgxpool.Pool, table string) bool {
	var n int

	err := p.QueryRow(context.Background(), "SELECT 1 FROM information_schema.tables WHERE table_name = $1", table).Scan(&n)

	return err == nil
}
