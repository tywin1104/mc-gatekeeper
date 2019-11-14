package server

import (
	"encoding/json"
	"net/http"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	"github.com/sirupsen/logrus"

	"github.com/dgrijalva/jwt-go"
	"github.com/spf13/viper"
)

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

var authMiddleware *jwtmiddleware.JWTMiddleware

// HandleAdminSignin handle auth token generation
func (svc *Service) HandleAdminSignin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds credentials
		err := json.NewDecoder(r.Body).Decode(&creds)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// check for valid admin login credentials
		if !(creds.Username == viper.GetString("adminUsername") && creds.Password == viper.GetString("adminPassword")) {
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
		tokenString, err := token.SignedString([]byte(viper.GetString("jwtTokenSecret")))
		if err != nil {
			// If there is an error in creating the JWT return an internal server error
			svc.logger.WithFields(logrus.Fields{
				"err": err.Error(),
			}).Error("Unable to sign the JWT token")
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

// GetAuthMiddleware return the auth middleware which verifys jwt auth token
func (svc *Service) GetAuthMiddleware() *jwtmiddleware.JWTMiddleware {
	if authMiddleware == nil {
		authMiddleware = jwtmiddleware.New(jwtmiddleware.Options{
			ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
				return []byte(viper.GetString("jwtTokenSecret")), nil
			},
			SigningMethod: jwt.SigningMethodHS256,
		})
	}
	return authMiddleware
}
