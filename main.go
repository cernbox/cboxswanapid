package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/cernbox/cboxswanapid/handlers"
	"github.com/cernbox/gohub/goconfig"
	"github.com/cernbox/gohub/gologger"
	"github.com/coreos/go-oidc/v3/oidc"
	gh "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// Build information obtained with the help of -ldflags
var (
	appName       string
	buildDate     string // date -u
	gitTag        string // git describe --exact-match HEAD
	gitNearestTag string // git describe --abbrev=0 --tags HEAD
	gitCommit     string // git rev-parse HEAD
)

var gc *goconfig.GoConfig

func init() {
	gc = goconfig.New()
	gc.SetConfigName("cboxswanapid")
	gc.AddConfigurationPaths("/etc/cboxswanapid")
	gc.Add("port", 2005, "Port to listen for connections")
	gc.Add("applog", "stderr", "File to log application data")
	gc.Add("httplog", "stderr", "File to log HTTP requests")
	gc.Add("secret", "changeme", "Shared secret with SWAN")
	gc.Add("signkey", "changeme", "Secret to sign JWT tokens")
	gc.Add("swanclient", "swan-service", "SWAN client id")
	gc.Add("oidcprovider", "https://auth.cern.ch/auth/realms/cern", "OIDC endpoint")
	gc.Add("allowfrom", "swan[a-z0-9-]*.cern.ch", "Check the Referer/Origin request header (depending on the endpoint) and return Bad Request if no match.")
	gc.Add("shibreferer", "https://login.cern.ch", "Shibolleth referer for /authenticate request.")
	gc.Add("cboxgroupdsecret", "", "Shared secret to communicate with the cboxgroupd daemon")
	gc.Add("cboxgroupdurl", "http://localhost:2002/api/v1/search", "URL to address the cboxgroupd daemon")
	gc.Add("config", "", "Configuration file to use")
	gc.Add("cboxsharescript", "/b/dev/kuba/devel.cernbox_utils/cernbox-swan-project", "Path to the cernbox share script")
	gc.Add("log-level", "info", "log level to use (debug, info, warn, error)")
	gc.BindFlags()
	gc.ReadConfig()
}

func main() {

	logger := gologger.New(gc.GetString("log-level"), gc.GetString("applog"))

	router := mux.NewRouter()

	ctx := context.Background()
	oidcProvider, err := oidc.NewProvider(ctx, gc.GetString("oidcprovider"))
	if err != nil {
		panic(fmt.Errorf("error configuring oidc provider", err))
	}
	var verifier = oidcProvider.Verifier(&oidc.Config{ClientID: gc.GetString("swanclient")})

	tokenHandler := handlers.CheckNothing(logger, handlers.Token(logger, gc.GetString("signkey"), gc.GetString("allowfrom"), gc.GetString("shibreferer")))
	tokenHandler2 := handlers.CheckOIDCToken(logger, ctx, verifier, handlers.Token2(logger, gc.GetString("signkey")), gc.GetString("allowfrom"))

	sharedHandler := handlers.CheckJWTToken(logger, gc.GetString("signkey"), handlers.Shared(logger, gc.GetString("cboxsharescript"), gc.GetString("allowfrom"), "list-shared-with", false))
	sharingHandler := handlers.CheckJWTToken(logger, gc.GetString("signkey"), handlers.Shared(logger, gc.GetString("cboxsharescript"), gc.GetString("allowfrom"), "list-shared-by", false))
	getIndividualShareHandler := handlers.CheckJWTToken(logger, gc.GetString("signkey"), handlers.Shared(logger, gc.GetString("cboxsharescript"), gc.GetString("allowfrom"), "list-shared-by", true))
	updateShareHandler := handlers.CheckJWTToken(logger, gc.GetString("signkey"), handlers.UpdateShare(logger, gc.GetString("cboxsharescript"), gc.GetString("allowfrom")))
	deleteShareHandler := handlers.CheckJWTToken(logger, gc.GetString("signkey"), handlers.DeleteShare(logger, gc.GetString("cboxsharescript"), gc.GetString("allowfrom")))
	searchHandler := handlers.CheckJWTToken(logger, gc.GetString("signkey"), handlers.Search(logger, gc.GetString("allowfrom"), gc.GetString("cboxgroupdurl"), gc.GetString("cboxgroupdsecret")))
	cloneShareHandler := handlers.CheckJWTToken(logger, gc.GetString("signkey"), handlers.CloneShare(logger, gc.GetString("cboxsharescript"), gc.GetString("allowfrom")))
	notFoundHandler := handlers.CheckJWTToken(logger, gc.GetString("signkey"), handlers.Handle404(logger))

	router.NotFoundHandler = notFoundHandler // default protection for non-existing resources is JWT

	router.Handle("/swanapi/v1/authenticate", tokenHandler).Methods("GET")
	router.Handle("/swanapi/v2/authenticate", tokenHandler2).Methods("GET")
	router.Handle("/swanapi/v1/shared", sharedHandler).Methods("GET")
	router.Handle("/swanapi/v1/sharing", sharingHandler).Methods("GET")
	router.Handle("/swanapi/v1/share", getIndividualShareHandler).Methods("GET")
	router.Handle("/swanapi/v1/share", updateShareHandler).Methods("PUT")
	router.Handle("/swanapi/v1/share", deleteShareHandler).Methods("DELETE")
	router.Handle("/swanapi/v1/search", searchHandler).Methods("GET")
	router.Handle("/swanapi/v1/clone", cloneShareHandler).Methods("POST")

	router.Handle("/swanapi/v1/shared", handlers.Options(logger, []string{"GET"}, gc.GetString("allowfrom"))).Methods("OPTIONS")
	router.Handle("/swanapi/v1/sharing", handlers.Options(logger, []string{"GET"}, gc.GetString("allowfrom"))).Methods("OPTIONS")
	router.Handle("/swanapi/v1/share", handlers.Options(logger, []string{"GET", "PUT", "DELETE"}, gc.GetString("allowfrom"))).Methods("OPTIONS")
	router.Handle("/swanapi/v1/clone", handlers.Options(logger, []string{"POST"}, gc.GetString("allowfrom"))).Methods("OPTIONS")
	router.Handle("/swanapi/v1/search", handlers.Options(logger, []string{"GET"}, gc.GetString("allowfrom"))).Methods("OPTIONS")

	out := getHTTPLoggerOut(gc.GetString("httplog"))
	loggedRouter := gh.LoggingHandler(out, router)

	logger.Info("server is listening", zap.Int("port", gc.GetInt("port")))
	logger.Warn("server stopped", zap.Error(http.ListenAndServe(fmt.Sprintf(":%d", gc.GetInt("port")), loggedRouter)))
}

func getHTTPLoggerOut(filename string) *os.File {
	if filename == "stderr" {
		return os.Stderr
	} else if filename == "stdout" {
		return os.Stdout
	} else {
		fd, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		return fd
	}
}

func showVersion() {
	// if gitTag is not empty we are on release build
	if gitTag != "" {
		fmt.Printf("%s %s commit:%s release-build\n", appName, gitNearestTag, gitCommit)
		os.Exit(0)
	}
	fmt.Printf("%s %s commit:%s dev-build\n", appName, gitNearestTag, gitCommit)
	os.Exit(0)
}
