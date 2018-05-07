package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	"strconv"

	"github.com/kayac/Gunfish"
	"github.com/sirupsen/logrus"
)

var version string

func main() {
	var (
		config      string
		enablePprof bool
		environment string
		logFormat   string
		logLevel    string
		logOutput   string
		port        int
		showVersion bool
	)

	flag.StringVar(&environment, "E", "production", "APNS environment. (production, development, or test)")
	flag.StringVar(&config, "c", "/etc/gunfish/config.toml", "specify config file.")
	flag.StringVar(&config, "config", "/etc/gunfish/config.toml", "specify config file.")
	flag.BoolVar(&enablePprof, "enable-pprof", false, ".")
	flag.StringVar(&environment, "environment", "production", "APNS environment. (production, development, or test)")
	flag.StringVar(&logFormat, "log-format", "", "specifies the log format: ltsv or json.")
	flag.StringVar(&logLevel, "log-level", "info", "set the log level (debug, warn, info)")
	flag.StringVar(&logOutput, "log-output", "", "set a string where you want to write the log message such as a file (specified path), 'stdout', 'stderr', or 'discard'. By default, leave it to 'github.com/sirupsen/logrus'.")
	flag.IntVar(&port, "port", 0, "Gunfish port number (range 1024-65535).")
	flag.BoolVar(&showVersion, "v", false, "show version number.")
	flag.BoolVar(&showVersion, "version", false, "show version number.")
	flag.Parse()

	if showVersion {
		fmt.Printf("Compiler: %s %s\n", runtime.Compiler, runtime.Version())
		fmt.Printf("Gunfish version: %s\n", version)
		return
	}

	initLogrus(logOutput, logFormat, logLevel)

	c, err := gunfish.LoadConfig(config)
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	c.Provider.DebugPort = 0
	if port != 0 {
		c.Provider.Port = port // Default port number
	}

	var env gunfish.Environment
	switch environment {
	case "production":
		env = gunfish.Production
	case "development":
		env = gunfish.Development
	case "test":
		env = gunfish.Test
	default:
		logrus.Error("Unknown environment: %s. Please look at help.", environment)
		os.Exit(1)
	}

	// for profiling
	if enablePprof {
		mux := http.NewServeMux()
		l, err := net.Listen("tcp", "localhost:0")
		if err != nil {
			logrus.Fatal(err)
		}
		debugAddr := l.Addr().String()
		_, p, err := net.SplitHostPort(debugAddr)
		if err != nil {
			logrus.Fatal(err)
		}
		dp, err := strconv.Atoi(p)
		if err != nil {
			logrus.Fatal(err)
		}
		logrus.Infof("Debug port (pprof) is %d.", dp)
		c.Provider.DebugPort = dp

		if enablePprof {
			mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
			mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
			mux.HandleFunc("/debug/pprof/", pprof.Index)
		}

		go func() {
			logrus.Fatal(http.Serve(l, mux))
		}()
	}

	gunfish.StartServer(c, env)
}

func initLogrus(logOutput string, format string, logLevel string) {
	switch format {
	case "ltsv":
		logrus.SetFormatter(&gunfish.LtsvFormatter{})
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}

	switch logOutput {
	case "":
		// do nothing
	case "stdout":
		logrus.SetOutput(os.Stdout)
	case "stderr":
		logrus.SetOutput(os.Stderr)
	case "discard":
		logrus.SetOutput(ioutil.Discard)
	default:
		file, err := os.OpenFile(logOutput, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			logrus.SetOutput(file)
		} else {
			logrus.Info("Failed to log to file, using default")
		}
	}

	lvl, err := logrus.ParseLevel(logLevel)
	if err != nil {
		lvl = logrus.InfoLevel
	}
	logrus.SetLevel(lvl)
}
