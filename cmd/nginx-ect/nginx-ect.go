package main

import (
	"flag"
	"os"
	"strings"
	"time"

	"github.com/robertlestak/nginx-ect/internal/nginx"
	"github.com/robertlestak/nginx-ect/internal/utils"
	log "github.com/sirupsen/logrus"
)

var (
	flags   = flag.NewFlagSet("nginx-ect", flag.ExitOnError)
	Version = "dev"
)

func init() {
	ll, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		ll = log.InfoLevel
	}
	log.SetLevel(ll)
}

func main() {
	l := log.WithFields(log.Fields{
		"app": "nginx-ect",
	})
	l.Debug("starting nginx-ect")
	logLevel := flags.String("l", log.GetLevel().String(), "log level")
	inFile := flags.String("i", "", "input file")
	diffFile := flags.String("d", "", "input file to diff against state file")
	stateFile := flags.String("s", "nginx-ect.state.json", "state file")
	ignoreServers := flags.String("x", "", "comma separated list of server names to exclude")
	verifyHash := flags.Bool("h", true, "verify hash")
	concurrency := flags.Int("c", 10, "concurrency")
	timeout := flags.String("t", "5s", "timeout")
	version := flags.Bool("v", false, "version")
	flags.Parse(os.Args[1:])
	ll, err := log.ParseLevel(*logLevel)
	if err != nil {
		ll = log.InfoLevel
	}
	log.SetLevel(ll)
	l = l.WithFields(log.Fields{
		"inFile":    *inFile,
		"diffFile":  *diffFile,
		"stateFile": *stateFile,
	})
	l.Debug("flags parsed")
	if *version {
		l.WithField("version", Version).Info("version")
		os.Exit(0)
	}
	if *inFile == "" && *diffFile == "" {
		l.Error("no input file specified")
		os.Exit(1)
	}
	var method string
	if *inFile != "" {
		method = "index"
	} else if *diffFile != "" {
		method = "diff"
		inFile = diffFile
	}
	var ignoreServersSlice []string
	if *ignoreServers != "" {
		ignoreServersSlice = strings.Split(*ignoreServers, ",")
		ignoreServersSlice = utils.CleanSlice(ignoreServersSlice)
	}
	nc := &nginx.NginxEct{
		ConfigFilePath: *inFile,
		StateFilePath:  *stateFile,
		IgnoreServers:  ignoreServersSlice,
		Concurrency:    *concurrency,
		VerifyHash:     *verifyHash,
	}
	pt, err := time.ParseDuration(*timeout)
	if err != nil {
		l.WithError(err).Error("failed to parse timeout")
		os.Exit(1)
	}
	nc.Timeout = pt
	switch method {
	case "index":
		l.Debug("indexing")
		if err := nc.Index(); err != nil {
			l.WithError(err).Error("errors during index")
			os.Exit(1)
		}
	case "diff":
		l.Debug("diffing")
		if err := nc.Diff(); err != nil {
			l.WithError(err).Error("errors during diff")
			os.Exit(1)
		}
	default:
		l.WithField("method", method).Error("invalid method")
		os.Exit(1)
	}
}
