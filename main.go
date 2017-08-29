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

func init() {
	flag.BoolVar(&fVersion, "version", false, "Show version")
	flag.IntVar(&fPort, "port", 2005, "Port to listen for connections")
	flag.StringVar(&fAppLog, "applog", "stderr", "File to log application data")
	flag.StringVar(&fHTTPLog, "httplog", "stderr", "File to log HTTP requests")
	flag.StringVar(&fSecret, "secret", "changeme", "Shared secret with SWAN")
	flag.StringVar(&fSignKey, "signkey", "changeme", "Secret to sign JWT tokens")
	flag.StringVar(&fAllowFrom, "allowfrom", "swan[a-z0-9-]*.cern.ch", "Check the Referer/Origin request header (depending on the endpoint) and return Bad Request if no match.")
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
	tokenHandler := handlers.CheckNothing(logger, handlers.Token(logger, fSignKey, fAllowFrom))
	sharedHandler := handlers.CheckJWTToken(logger, fSignKey, handlers.Shared(logger))

	notFoundHandler :=  handlers.CheckJWTToken(logger, fSignKey, handlers.Handle404(logger))

	router.NotFoundHandler = notFoundHandler // all path are protected by token except below:

	router.Handle("/swanapi/v1/authenticate", tokenHandler).Methods("GET")
	router.Handle("/swanapi/v1/shared", sharedHandler).Methods("GET")

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
