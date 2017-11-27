package main

import (
	"flag"
	"fmt"
	"github.com/cernbox/cboxswanapid/handlers"
	gh "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
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

func init() {
	flag.BoolVar(&fVersion, "version", false, "Show version")

	viper.SetDefault("port", 2005)
	viper.SetDefault("applog", "stderr")
	viper.SetDefault("httplog", "stderr")
	viper.SetDefault("secret", "changeme")
	viper.SetDefault("signkey", "changeme")
	viper.SetDefault("allowfrom", "swan[a-z0-9-]*.cern.ch")
	viper.SetDefault("shibreferer", "https://login.cern.ch")
	viper.SetDefault("cboxgroupdurl", "http://localhost:2002/api/v1/search")
	viper.SetDefault("cboxgroupdsecret", "changeme")

	viper.SetConfigName("cboxswanapid")
	viper.AddConfigPath("/etc/cboxswanapid/")
	flag.Int("port", 2005, "Port to listen for connections")
	flag.String("applog", "stderr", "File to log application data")
	flag.String("httplog", "stderr", "File to log HTTP requests")
	flag.String("secret", "changeme", "Shared secret with SWAN")
	flag.String("signkey", "changeme", "Secret to sign JWT tokens")
	flag.String("allowfrom", "swan[a-z0-9-]*.cern.ch", "Check the Referer/Origin request header (depending on the endpoint) and return Bad Request if no match.")
	flag.String("shibreferer", "https://login.cern.ch", "Shibolleth referer for /authenticate request.")
	flag.String("cboxgroupdsecret", "", "Shared secret to communicate with the cboxgroupd daemon")
	flag.String("cboxgroupdurl", "http://localhost:2002/api/v1/search", "URL to address the cboxgroupd daemon")
	flag.String("config", "", "Configuration file to use")
	flag.Parse()

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)
}

func main() {

	if fVersion {
		showVersion()
	}

	if viper.GetString("config") != "" {
		viper.SetConfigFile(viper.GetString("config"))
	}

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("error reading config file: %s", err))
	}

	config := zap.NewProductionConfig()
	config.OutputPaths = []string{viper.GetString("applog")}
	logger, _ := config.Build()

	router := mux.NewRouter()

	tokenHandler := handlers.CheckNothing(logger, handlers.Token(logger, viper.GetString("signkey"), viper.GetString("allowfrom"), viper.GetString("shibreferer")))

	sharedHandler := handlers.CheckJWTToken(logger, viper.GetString("signkey"), handlers.Shared(logger, viper.GetString("allowfrom"), "swan-list-projects-shared-with", false))
	sharingHandler := handlers.CheckJWTToken(logger, viper.GetString("signkey"), handlers.Shared(logger, viper.GetString("allowfrom"), "swan-list-projects-shared-by", false))
	getIndividualShareHandler := handlers.CheckJWTToken(logger, viper.GetString("signkey"), handlers.Shared(logger, viper.GetString("allowfrom"), "swan-list-projects-shared-by", true))
	updateShareHandler := handlers.CheckJWTToken(logger, viper.GetString("signkey"), handlers.UpdateShare(logger, viper.GetString("allowfrom")))
	deleteShareHandler := handlers.CheckJWTToken(logger, viper.GetString("signkey"), handlers.DeleteShare(logger, viper.GetString("allowfrom")))
	searchHandler := handlers.CheckJWTToken(logger, viper.GetString("signkey"), handlers.Search(logger, viper.GetString("cboxgroupdurl"), viper.GetString("cboxgroupdsecret")))
	cloneShareHandler := handlers.CheckJWTToken(logger, viper.GetString("signkey"), handlers.CloneShare(logger, viper.GetString("allowfrom")))
	notFoundHandler := handlers.CheckJWTToken(logger, viper.GetString("signkey"), handlers.Handle404(logger))

	router.NotFoundHandler = notFoundHandler // default protection for non-existing resources is JWT

	router.Handle("/swanapi/v1/authenticate", tokenHandler).Methods("GET")
	router.Handle("/swanapi/v1/shared", sharedHandler).Methods("GET")
	router.Handle("/swanapi/v1/sharing", sharingHandler).Methods("GET")
	router.Handle("/swanapi/v1/share", getIndividualShareHandler).Methods("GET")
	router.Handle("/swanapi/v1/share", updateShareHandler).Methods("PUT")
	router.Handle("/swanapi/v1/share", deleteShareHandler).Methods("DELETE")
	router.Handle("/swanapi/v1/search/{filter}", searchHandler).Methods("GET")

	router.Handle("/swanapi/v1/shared", handlers.Options(logger, []string{"GET"}, viper.GetString("allowfrom"))).Methods("OPTIONS")
	router.Handle("/swanapi/v1/sharing", handlers.Options(logger, []string{"GET"}, viper.GetString("allowfrom"))).Methods("OPTIONS")
	router.Handle("/swanapi/v1/share", handlers.Options(logger, []string{"GET", "PUT", "DELETE"}, viper.GetString("allowfrom"))).Methods("OPTIONS")

	router.Handle("/swanapi/v1/clone", cloneShareHandler).Methods("POST")
	router.Handle("/swanapi/v1/clone", handlers.Options(logger, []string{"POST"}, viper.GetString("allowfrom"))).Methods("OPTIONS")

	out := getHTTPLoggerOut(viper.GetString("httplog"))
	loggedRouter := gh.LoggingHandler(out, router)

	logger.Info("server is listening", zap.Int("port", viper.GetInt("port")))
	logger.Warn("server stopped", zap.Error(http.ListenAndServe(fmt.Sprintf(":%d", viper.GetInt("port")), loggedRouter)))
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
