package auth

import (
	"encoding/json"
	"net/http"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	"github.com/tywin1104/mc-whitelist/config"

	"github.com/dgrijalva/jwt-go"
)

var users = map[string]string{
	"user1": "password1",
	"user2": "password2",
}

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

// AdminSigninHandler handles admin login request and generate jwt token if credentials are valid
func AdminSigninHandler(c *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds credentials
		err := json.NewDecoder(r.Body).Decode(&creds)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		expectedPassword, ok := users[creds.Username]

		if !ok || expectedPassword != creds.Password {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Declare the expiration time of the token
		expirationTime := time.Now().Add(20 * time.Minute)
		// Create the JWT claims, which includes the username and expiry time
		claims := &claims{
			Username: creds.Username,
			StandardClaims: jwt.StandardClaims{
				// In JWT, the expiry time is expressed as unix milliseconds
				ExpiresAt: expirationTime.Unix(),
			},
		}

		// Declare the token with the algorithm used for signing, and the claims
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		// Create the JWT string
		tokenString, err := token.SignedString(c.JWTTokenSecret)
		if err != nil {
			// If there is an error in creating the JWT return an internal server error
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		msg := map[string]map[string]interface{}{"token": {
			"value":   tokenString,
			"expires": expirationTime,
		}}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(msg)
	}
}

// AuthMiddleware checks for valid attached to API routes that are not publicly accessible
var AuthMiddleware = jwtmiddleware.New(jwtmiddleware.Options{
	ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
		return []byte("my_secret_key"), nil
	},
	SigningMethod: jwt.SigningMethodHS256,
})

// func fromCookie(accessTokenName string) jwtmiddleware.TokenExtractor {
// 	return func(r *http.Request) (string, error) {
// 		cookie, _ := r.Cookie(accessTokenName)
// 		fmt.Println(cookie)
// 		if cookie != nil {
// 			return cookie.Value, nil
// 		}
// 		return "", nil
// 	}
// }
