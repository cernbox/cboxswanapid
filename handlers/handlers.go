package handlers

import (
	"bytes"
	"github.com/gorilla/context"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
	"time"
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
type CmdError struct {
	Error      string `json:"error"`
	Statuscode int    `json:"statuscode"`
}

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

func CheckHostAllowed(origin url.URL, allowFrom string, logger *zap.Logger) bool {

	if origin.Scheme != "https" {
		logger.Info(fmt.Sprintf("***** Only https scheme is supported. Origin is %s", origin))
		return false
	}

	// TODO: case insensitive
	matched, _ := regexp.MatchString(allowFrom, origin.Host)

	logger.Info(fmt.Sprintf("***** Checking Allowed Host:  %s matches %s => %s", origin, allowFrom, matched))

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

		m, err := url.ParseQuery(r.URL.RawQuery)

		if err != nil {
			logger.Error(fmt.Sprintf("URL query parsing error: %s '%s' ", err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var origin string

		if val, ok := m["Origin"]; ok {
			origin = val[0]
		} else {
			logger.Error(fmt.Sprintf("URL missing origin query parameter"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		referer, err := url.Parse(origin)

		if err != nil {
			logger.Error(fmt.Sprintf("Error parsing Referer header: '%s' %s", r.Header.Get("Referer"), err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		referer_url := url.URL{Scheme: referer.Scheme, Host: referer.Host}
		referer_host := referer_url.String() // format the allowed host including the scheme

		// TODO(labkode): check with kuba
		if referer_host == shibReferer {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if !CheckHostAllowed(*referer, allowFrom, logger) {
			logger.Error(fmt.Sprintf("Referer host '%s' does not match allowFrom pattern '%s'", referer.Host, allowFrom))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		//logger.Info(fmt.Sprintf("***** ALLOWED_HOST: %s",referer_host))

		expire := time.Now().Add(time.Duration(3600) * time.Second) // TODO(labkode): expire data in config

		token := jwt.New(jwt.GetSigningMethod("HS256"))
		claims := token.Claims.(jwt.MapClaims)
		claims["username"] = username
		claims["exp"] = expire.UnixNano()
		tokenString, _ := token.SignedString([]byte(signKey))

		response := &struct {
			Token  string    `json:"authtoken"`
			Expire time.Time `json:"expire"`
		}{Token: tokenString, Expire: expire}

		jsonBody, _ := json.Marshal(response)

		w.Header().Set("X-Frame-Options", fmt.Sprintf("ALLOW-FROM %s", referer_host))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<script>parent.postMessage(" + string(jsonBody) + ", '" + referer_host + "');</script>"))
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

		context.Set(r, "username", username)
		fmt.Println(r.URL)
		handler.ServeHTTP(w, r)
	})
}

func Handle404(logger *zap.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		return
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

	origin, err := url.Parse(r.Header.Get("Origin"))

	if err != nil {
		logger.Error(fmt.Sprintf("Error parsing Origin header: '%s' %s", r.Header.Get("Origin"), err))
		w.WriteHeader(http.StatusBadRequest)
		return false
	}

	if !CheckHostAllowed(*origin, allowFrom, logger) {
		logger.Error(fmt.Sprintf("Origin URL '%s' does not match allowFrom pattern '%s'", origin, allowFrom))
		w.WriteHeader(http.StatusBadRequest)
		return false
	}

	w.Header().Set("Access-Control-Allow-Origin", origin.String())
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

func Options(logger *zap.Logger, allowedMethods []string, allowFrom string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if !CORSProcessOriginHeader(logger, w, r, allowFrom) {
			return
		}

		x := r.Header.Get("Access-Control-Request-Headers")

		if strings.ToUpper(x) != "AUTHORIZATION" {
			logger.Error(fmt.Sprintf("OPTIONS: Wrong or missing Access-Control-Request-Headers header: '%s'", x))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		x = r.Header.Get("Access-Control-Request-Method")

		if !stringInSlice(strings.ToUpper(x), allowedMethods) {
			logger.Error(fmt.Sprintf("OPTIONS: Wrong or missing Access-Control-Request-Method header: '%' ", x))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ","))
		w.Header().Set("Access-Control-Allow-Headers", "Authorization")

	})
}

/* ------------------------ */

func executeCMD(cmd *exec.Cmd) (*bytes.Buffer, *bytes.Buffer, error) {

	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	cmd.Stdout = outBuf
	cmd.Stderr = errBuf
	err := cmd.Run()

	return outBuf, errBuf, err
}

func Search(logger *zap.Logger, cboxgroupdUrl, cboxgroupdSecret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filter := mux.Vars(r)["filter"]
		fmt.Printf("filter:%s\n", filter)

		url := strings.Join([]string{cboxgroupdUrl, filter}, "/")
		req, err := http.NewRequest("GET", url, nil)
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", cboxgroupdSecret))
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			logger.Error(fmt.Sprintf("error sending request: %s", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(res.StatusCode)
		defer res.Body.Close()
		io.Copy(w, res.Body)
	})
}

func CloneShare(logger *zap.Logger, allowFrom string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if !CORSProcessOriginHeader(logger, w, r, allowFrom) {
			return
		}

		v := context.Get(r, "username")
		username, _ := v.(string)

		logger.Info("loggedin user is " + username)

		sharer := ""
		shared_project := ""
		cloned_project := ""

		m, err := url.ParseQuery(r.URL.RawQuery)

		if err != nil {
			logger.Error(fmt.Sprintf("URL query parsing error: %s '%s' ", err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if val, ok := m["sharer"]; ok {
			sharer = val[0]
		} else {
			logger.Error(fmt.Sprintf("URL missing query parameter: sharer not specified"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if val, ok := m["project"]; ok {
			shared_project = val[0]
		} else {
			logger.Error(fmt.Sprintf("URL missing query parameter: project to clone not specified"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if val, ok := m["destination"]; ok {
			cloned_project = val[0]
		} else {
			logger.Error(fmt.Sprintf("URL missing query parameter: new name of the cloned project not specified"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		args := []string{"-c", "/root/kuba-config.php", "--json", "clone-share", sharer, shared_project, username, cloned_project}

		logger.Info(fmt.Sprintf("cmd args %s", args))

		cmd := exec.Command("/b/dev/kuba/devel.cernbox_utils/cernbox-swan-project", args...)

		jsonResponse, errBuf, err := executeCMD(cmd)

		if err != nil {

			logger.Error(fmt.Sprintf("Error calling cmd %s %s %s: '%s'", cmd.Path, cmd.Args, err, errBuf.String()))

			cmderr := CmdError{Statuscode: http.StatusInternalServerError}
			json.Unmarshal(jsonResponse.Bytes(), &cmderr)

			// TODO: inject error string if applicable
			w.WriteHeader(cmderr.Statuscode)
			//return
		}

		w.Write(jsonResponse.Bytes())

	})
}

func DeleteShare(logger *zap.Logger, allowFrom string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if !CORSProcessOriginHeader(logger, w, r, allowFrom) {
			return
		}

		v := context.Get(r,"username")
		username, _ := v.(string)

		logger.Info("loggedin user is " + username)

		project := ""

		m, err := url.ParseQuery(r.URL.RawQuery)

		if err != nil {
			logger.Error(fmt.Sprintf("URL query parsing error: %s '%s' ", err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if val, ok := m["project"]; ok {
			project = val[0]
		} else {
			logger.Error(fmt.Sprintf("URL missing query parameter: project not specified"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		args := []string{"-c", "/root/kuba-config.php", "--json", "swan-delete-project-share", username, project}

		logger.Info(fmt.Sprintf("cmd args %s", args))

		cmd := exec.Command("/b/dev/kuba/devel.cernbox_utils/cernbox-swan-project", args...)

		jsonResponse, errBuf, err := executeCMD(cmd)

		if err != nil {

			logger.Error(fmt.Sprintf("Error calling cmd %s %s %s: '%s'", cmd.Path, cmd.Args, err, errBuf.String()))

			cmderr := CmdError{Statuscode: http.StatusInternalServerError}
			json.Unmarshal(jsonResponse.Bytes(), &cmderr)

			// TODO: inject error string if applicable
			w.WriteHeader(cmderr.Statuscode)
			//return
		}

		w.Write(jsonResponse.Bytes())

	})
}

func UpdateShare(logger *zap.Logger, allowFrom string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if !CORSProcessOriginHeader(logger, w, r, allowFrom) {
			return
		}

		v := context.Get(r,"username")
		username, _ := v.(string)

		logger.Info("loggedin user is " + username)

		project := ""

		m, err := url.ParseQuery(r.URL.RawQuery)

		if err != nil {
			logger.Error(fmt.Sprintf("URL query parsing error: %s '%s' ", err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if val, ok := m["project"]; ok {
			project = val[0]
		} else {
			logger.Error(fmt.Sprintf("URL missing query parameter: project not specified"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		args := []string{"-c", "/root/kuba-config.php", "--json", "update-share", username, project}

		type Sharee struct {
			Name   string `json:"name"`   // name of user or group
			Entity string `json:"entity"` // "u" is user, "egroup" is group
		}

		type ShareRequest struct {
			ShareWith []Sharee `json:"share_with"`
		}

		var share_request ShareRequest

		if err = json.NewDecoder(r.Body).Decode(&share_request); err != nil {
			logger.Error(fmt.Sprintf("Cannot unmarshal JSON request body"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// TODO: BadRequest if empty ShareWith array

		if len(share_request.ShareWith) == 0 {
			logger.Error(fmt.Sprintf("Empty request"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		logger.Info(fmt.Sprintf("request %s", share_request))

		for i := range share_request.ShareWith {
			// FIXME: TODO: sanitize names
			// FIXME: TODO: check for missing fields, e.g. empty name or empty entity
			share := share_request.ShareWith[i]
			args = append(args, share.Entity+":"+share.Name)
		}

		logger.Info(fmt.Sprintf("cmd args %s", args))

		cmd := exec.Command("/b/dev/kuba/devel.cernbox_utils/cernbox-swan-project", args...)

		jsonResponse, errBuf, err := executeCMD(cmd)

		if err != nil {

			logger.Error(fmt.Sprintf("Error calling cmd %s %s %s: '%s'", cmd.Path, cmd.Args, err, errBuf.String()))

			cmderr := CmdError{Statuscode: http.StatusInternalServerError}
			json.Unmarshal(jsonResponse.Bytes(), &cmderr)

			// TODO: inject error string if applicable
			w.WriteHeader(cmderr.Statuscode)
			//return
		}

		w.Write(jsonResponse.Bytes())

	})
}

func Shared(logger *zap.Logger, allowFrom string, action string, requireProject bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if !CORSProcessOriginHeader(logger, w, r, allowFrom) {
			return
		}

		v := context.Get(r,"username")
		username, _ := v.(string)

		logger.Info("loggedin user is " + username)

		project := ""

		if requireProject {

			m, err := url.ParseQuery(r.URL.RawQuery)

			if err != nil {
				logger.Error(fmt.Sprintf("URL query parsing error: %s '%s' ", err))
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if val, ok := m["project"]; ok {
				project = val[0]
			} else {
				logger.Error(fmt.Sprintf("URL missing query parameter: project not specified "))
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}

		type shareInfo struct {
			User string `json:"user"`
			Path string `json:"path"`
			Size int    `json:"size"`
			Date int64  `json:"date"`
		}

		type response struct {
			Shared []*shareInfo `json:"shared"`
		}

		args := []string{"-c", "/root/kuba-config.php", "--json", action}

		if project != "" {
			args = append(args, "--project", project)
		}

		args = append(args, username)

		logger.Info(fmt.Sprintf("cmd args %s", args))

		cmd := exec.Command("/b/dev/kuba/devel.cernbox_utils/cernbox-swan-project", args...)

		jsonResponse, errBuf, err := executeCMD(cmd)

		if err != nil {

			logger.Error(fmt.Sprintf("Error calling cmd %s %s %s: '%s' ", cmd.Path, cmd.Args, err, errBuf.String()))

			cmderr := CmdError{Statuscode: http.StatusInternalServerError}
			json.Unmarshal(jsonResponse.Bytes(), &cmderr)

			// TODO: inject error string if applicable
			w.WriteHeader(cmderr.Statuscode)
			//return
		}

		w.Write(jsonResponse.Bytes())
	})
}
