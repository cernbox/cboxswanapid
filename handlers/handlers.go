package handlers

import (
	"context"
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
//	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"net/http"
        "net/url"
	"strings"
	"time"
	"regexp"
	"fmt"
)

/////////////////
/// for debugging
/////////////////
// formatRequest generates ascii representation of a request
func formatRequest(r *http.Request) string {
 // Create return string
 var request []string

 // Add the request string
 url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
 request = append(request, url)

 // Add the host
 request = append(request, fmt.Sprintf("Host: %v", r.Host))

 // Loop through headers
 for name, headers := range r.Header {
   name = strings.ToLower(name)
   for _, h := range headers {
     request = append(request, fmt.Sprintf("%v: %v", name, h))
   }
 }
 
 // If this is a POST, add post data
 if r.Method == "POST" {
    r.ParseForm()
    request = append(request, "\n")
    request = append(request, r.Form.Encode())
 } 

  // Return the request as a string
  return strings.Join(request, "\n")
}


/////////////////


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

func CheckNothing(logger *zap.Logger, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	handler.ServeHTTP(w, r)
	})
}


func CheckHostAllowed(origin string, allowFrom string) bool {
     
     // TODO: case insensitive
     matched,_ := regexp.MatchString(allowFrom,origin)

     return matched

}

func Token(logger *zap.Logger, signKey string, allowFrom string, shibReferer string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	        logger.Info(formatRequest(r))

		username := r.Header.Get("adfs_login") // this comes back from shibolleth (the name of the header depends on shibd configuration)

		if username == "" {
			logger.Error("Request header 'adfs_login' is empty or not set")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		referer,err := url.Parse(r.Header.Get("Referer"))

		if err != nil {
		   logger.Error(fmt.Sprintf("Error parsing Referer header: '%s' %s",r.Header.Get("Referer"), err))
		   w.WriteHeader(http.StatusBadRequest)
		   return
		}

		referer_url := url.URL{Scheme:referer.Scheme, Host:referer.Host}
		referer_host := referer_url.String(); // format the allowed host including the scheme

		if referer_host == shibReferer {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if !CheckHostAllowed(referer_host, allowFrom) {
		       logger.Error(fmt.Sprintf("Referer host '%s' does not match allowFrom pattern '%s'",referer.Host,allowFrom))
		       w.WriteHeader(http.StatusBadRequest)
		       return
		       }

		//logger.Info(fmt.Sprintf("***** ALLOWED_HOST: %s",referer_host))

		expire := time.Now().Add(time.Duration(3600)*time.Second) // TODO(labkode): expire data in config

		token := jwt.New(jwt.GetSigningMethod("HS256"))
		claims := token.Claims.(jwt.MapClaims)
		claims["username"] = username
		claims["exp"] = expire.UnixNano() 
		tokenString, _ := token.SignedString([]byte(signKey))

		response := &struct {
			Token string `json:"authtoken"`
			Expire time.Time `json:"expire"`
		}{Token: tokenString, Expire: expire}

		jsonBody, _ := json.Marshal(response)

		w.Header().Set("X-Frame-Options", fmt.Sprintf("ALLOW-FROM %s",referer_host))
		w.Write([]byte("<script>parent.postMessage(" + string(jsonBody) + ", '" + referer_host + "');</script>"))
		w.WriteHeader(http.StatusOK)
	})
}

func CheckJWTToken(logger *zap.Logger, signKey string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The jwt token is passed in the header: Authorization: Bearer mysecret
		h := r.Header.Get("Authorization")
		parts := strings.Split(h, " ")
		if len(parts) != 2 {
			logger.Error("wrong JWT header")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		token := parts[1]

		rawToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			return []byte(signKey), nil
		})
		if err != nil {
			logger.Error("invalid JWT token")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		claims := rawToken.Claims.(jwt.MapClaims)
		username, ok := claims["username"].(string)
		if !ok {
			logger.Error("jwt token username claim is not a string")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(context.Background(), "username", username)
		r = r.WithContext(ctx)

		handler.ServeHTTP(w, r)
	})
}

func Handle404(logger *zap.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	          w.WriteHeader(http.StatusNotFound)
	  })
     }


func Handle200(logger *zap.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	          logger.Info(formatRequest(r))
	          //w.WriteHeader(http.Success)
	  })
     }

// Handle CORS Origin header and return true if the request is allowed to continue
func CORSProcessOriginHeader(logger *zap.Logger, w http.ResponseWriter, r *http.Request, allowFrom string) bool {

     	origin,err := url.Parse(r.Header.Get("Origin"))

	if err != nil {
		logger.Error(fmt.Sprintf("Error parsing Origin header: '%s' %s",r.Header.Get("Origin"), err))
		 w.WriteHeader(http.StatusBadRequest)
		 return false
	}

	x := url.URL{Scheme:origin.Scheme, Host:origin.Host}
	origin_url := x.String(); // format the allowed host including the scheme

	if !CheckHostAllowed(origin_url, allowFrom) {
	       logger.Error(fmt.Sprintf("Origin URL '%s' does not match allowFrom pattern '%s'",origin.Host,allowFrom))
	       w.WriteHeader(http.StatusBadRequest)
	       return false
	       }

	w.Header().Set("Access-Control-Allow-Origin",origin_url)
	return true
}


func stringInSlice(str string, list []string) bool {
 	for _, v := range list {
 		if v == str {
 			return true
 		}
 	}
 	return false
 }


func Options(logger *zap.Logger, allowedMethods []string, allowFrom string ) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	if !CORSProcessOriginHeader(logger,w,r,allowFrom) { 
	   return 
	}

	x := r.Header.Get("Access-Control-Request-Headers")

	if strings.ToUpper(x) != "AUTHORIZATION" {
	       logger.Error(fmt.Sprintf("OPTIONS: Wrong or missing Access-Control-Request-Headers header: '%s'",x))
	       w.WriteHeader(http.StatusBadRequest)
	       return
	}

	x = r.Header.Get("Access-Control-Request-Method")

	if ! stringInSlice(strings.ToUpper(x),allowedMethods) {
	       logger.Error(fmt.Sprintf("OPTIONS: Wrong or missing Access-Control-Request-Method header: '%' ",x))
	       w.WriteHeader(http.StatusBadRequest)
	       return
	}

	w.Header().Set("Access-Control-Allow-Methods",strings.Join(allowedMethods,","))
	w.Header().Set("Access-Control-Allow-Headers","Authorization")

})
}

func Shared(logger *zap.Logger, allowFrom string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	        
		if !CORSProcessOriginHeader(logger,w,r,allowFrom) { 
	   	  return 
		}

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

