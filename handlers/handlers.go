package handlers

import (
	"context"
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"time"
)

func CheckSharedSecret(logger *zap.Logger, secret string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The secret is passed in the header: Authorization: Bearer mysecret
		h := r.Header.Get("Authorization")
		secret := "bearer " + secret
		if secret != strings.ToLower(h) {
			logger.Warn("wrong secret")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	})
}

func Token(logger *zap.Logger, signKey string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := mux.Vars(r)["username"]
		if username == "" {
			logger.Error("username is empty")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		token := jwt.New(jwt.GetSigningMethod("HS256"))
		claims := token.Claims.(jwt.MapClaims)
		claims["username"] = username
		claims["exp"] = time.Now().Add(time.Second * 3600).UnixNano() // TODO(labkode): expire data in config
		tokenString, _ := token.SignedString([]byte(signKey))

		response := &struct {
			Token string `json:"token"`
		}{Token: tokenString}

		jsonBody, _ := json.Marshal(response)

		w.WriteHeader(http.StatusOK)
		w.Header().Set("X-Frame-Options", "ALLOW-FROM  swan001.cern.ch")
		w.Write([]byte("<script>parent.postMessage(" + string(jsonBody) + ", 'swan001.cern.ch');</script>"))
		w.Write(jsonBody)
	})
}

func CheckJWTToken(logger *zap.Logger, signKey string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The jwt token is passed in the header: Authorization: Bearer mysecret
		h := r.Header.Get("Authorization")
		parts := strings.Split(h, " ")
		if len(parts) != 2 {
			logger.Error("wrong JWT header")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		token := parts[1]

		rawToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			return []byte(signKey), nil
		})
		if err != nil {
			logger.Error("invalid JWT token")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		claims := rawToken.Claims.(jwt.MapClaims)
		username, ok := claims["username"].(string)
		if !ok {
			logger.Error("jwt token username claim is not a string")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(context.Background(), "username", username)
		r = r.WithContext(ctx)

		handler.ServeHTTP(w, r)
	})
}

func Shared(logger *zap.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := r.Context().Value("username")
		username, _ := v.(string)

		logger.Info("loggedin user is " + username)
		w.Write([]byte(username))

		type shareInfo struct {
			User string `json:"user"`
			Path string `json:"path"`
			Size int    `json:"size"`
			Date int64  `json:"date"`
		}

		type response struct {
			Shared []*shareInfo `json:"shared"`
		}

		res := &response{
			Shared: []*shareInfo{
				&shareInfo{
					User: username,
					Path: "Swan projects/project 1",
					Size: 129399,
					Date: time.Now().UnixNano(),
				},
				&shareInfo{
					User: username,
					Path: "Swan projects/project 2",
					Size: 12939999,
					Date: time.Now().UnixNano(),
				},
			},
		}

		jsonResponse, _ := json.Marshal(res)
		w.Write(jsonResponse)
	})
}
