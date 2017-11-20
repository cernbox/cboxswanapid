package main

import (
	"flag"
	"fmt"
	"github.com/cernbox/cboxswanapid/handlers"
	gh "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
)

// Build information obtained with the help of -ldflags
var (
	appName       string
	buildDate     string // date -u
	gitTag        string // git describe --exact-match HEAD
	gitNearestTag string // git describe --abbrev=0 --tags HEAD
	gitCommit     string // git rev-parse HEAD
)

var fVersion bool
var fPort int
var fAppLog string
var fHTTPLog string
var fSignKey string
var fSecret string
var fAllowFrom string;
var fShibReferer string;

func init() {
	flag.BoolVar(&fVersion, "version", false, "Show version")
	flag.IntVar(&fPort, "port", 2005, "Port to listen for connections")
	flag.StringVar(&fAppLog, "applog", "stderr", "File to log application data")
	flag.StringVar(&fHTTPLog, "httplog", "stderr", "File to log HTTP requests")
	flag.StringVar(&fSecret, "secret", "changeme", "Shared secret with SWAN")
	flag.StringVar(&fSignKey, "signkey", "changeme", "Secret to sign JWT tokens")
	flag.StringVar(&fAllowFrom, "allowfrom", "swan[a-z0-9-]*.cern.ch", "Check the Referer/Origin request header (depending on the endpoint) and return Bad Request if no match.")
	flag.StringVar(&fShibReferer, "shibreferer", "https://login.cern.ch", "Shibolleth referer for /authenticate request.")
	flag.Parse()
}

func main() {

	if fVersion {
		showVersion()
	}

	config := zap.NewProductionConfig()
	config.OutputPaths = []string{fAppLog}
	logger, _ := config.Build()

	router := mux.NewRouter()

	//tokenHandler := handlers.CheckSharedSecret(logger, fSecret, handlers.Token(logger, fSignKey))
	tokenHandler := handlers.CheckNothing(logger, handlers.Token(logger, fSignKey, fAllowFrom, fShibReferer))

	sharedHandler := handlers.CheckJWTToken(logger, fSignKey, handlers.Shared(logger,fAllowFrom,"list-shared-with",false))
	sharingHandler := handlers.CheckJWTToken(logger, fSignKey, handlers.Shared(logger,fAllowFrom,"list-shared-by",false))
	getIndividualShareHandler := handlers.CheckJWTToken(logger, fSignKey, handlers.Shared(logger,fAllowFrom,"list-shared-by",true))
	updateShareHandler := handlers.CheckJWTToken(logger, fSignKey, handlers.UpdateShare(logger,fAllowFrom))
	deleteShareHandler := handlers.CheckJWTToken(logger, fSignKey, handlers.DeleteShare(logger,fAllowFrom))

	cloneShareHandler := handlers.CheckJWTToken(logger, fSignKey, handlers.CloneShare(logger,fAllowFrom))

	notFoundHandler :=  handlers.CheckJWTToken(logger, fSignKey, handlers.Handle404(logger))

	router.NotFoundHandler = notFoundHandler // default protection for non-existing resources is JWT

	router.Handle("/swanapi/v1/authenticate", tokenHandler).Methods("GET")
	router.Handle("/swanapi/v1/shared", sharedHandler).Methods("GET")
	router.Handle("/swanapi/v1/sharing", sharingHandler).Methods("GET")
	router.Handle("/swanapi/v1/share", getIndividualShareHandler).Methods("GET")
	router.Handle("/swanapi/v1/share", updateShareHandler).Methods("PUT")
	router.Handle("/swanapi/v1/share", deleteShareHandler).Methods("DELETE")

	router.Handle("/swanapi/v1/shared", handlers.Options(logger,[]string{"GET"},fAllowFrom)).Methods("OPTIONS")
	router.Handle("/swanapi/v1/sharing", handlers.Options(logger,[]string{"GET"},fAllowFrom)).Methods("OPTIONS")
	router.Handle("/swanapi/v1/share", handlers.Options(logger,[]string{"GET","PUT","DELETE"},fAllowFrom)).Methods("OPTIONS")

	router.Handle("/swanapi/v1/clone", cloneShareHandler).Methods("POST")
	router.Handle("/swanapi/v1/clone", handlers.Options(logger,[]string{"POST"},fAllowFrom)).Methods("OPTIONS")


	out := getHTTPLoggerOut(fHTTPLog)
	loggedRouter := gh.LoggingHandler(out, router)

	logger.Info("server is listening", zap.Int("port", fPort))
	logger.Warn("server stopped", zap.Error(http.ListenAndServe(fmt.Sprintf(":%d", fPort), loggedRouter)))
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
