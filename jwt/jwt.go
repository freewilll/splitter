package jwt

import (
	"log"
	"net/http"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

var expirationTime = 30 * time.Minute

var jwtKey = []byte("my-secret-stream-key")

type claims struct {
	UserID int `json:"user_id"`
	jwt.StandardClaims
}

// CreateCookie creates an cookie containing a JWT token that is set to expire in
// expirationTime.
func CreateCookie(userID int, cookieName string) http.Cookie {
	expirationTime := time.Now().Add(expirationTime)

	// Create a claim with an expiry and userID
	claims := &claims{
		UserID: userID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	// Create the JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		panic(err)
	}

	// Return an http cookie with the token
	return http.Cookie{
		Name:    cookieName,
		Value:   tokenString,
		Expires: expirationTime,
	}
}

// VerifyToken verifies a JWT token. If successful, the function returns (userID, true),
// if unsuccessful, it returns (0, false)
func VerifyToken(tokenString string) (int, bool) {
	claims := &claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			log.Println("Invalid signature")
			return 0, false
		}
		log.Println("Bad jwt token")
		return 0, false
	}

	if !token.Valid {
		log.Println("Invalid jwt token")
		return 0, false
	}

	return claims.UserID, true
}
