package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	"strconv"

	gunfish "github.com/kayac/Gunfish"
	"github.com/kayac/Gunfish/apns"
	"github.com/kayac/Gunfish/config"
	"github.com/sirupsen/logrus"
)

var version string

func main() {
	var (
		confPath    string
		environment string
		logFormat   string
		port        int
		enablePprof bool
		showVersion bool
		logLevel    string
	)

	flag.StringVar(&confPath, "config", "/etc/gunfish/config.toml", "specify config file.")
	flag.StringVar(&confPath, "c", "/etc/gunfish/config.toml", "specify config file.")
	flag.StringVar(&environment, "environment", "production", "APNS environment. (production, development, or test)")
	flag.StringVar(&environment, "E", "production", "APNS environment. (production, development, or test)")
	flag.IntVar(&port, "port", 0, "Gunfish port number (range 1024-65535).")
	flag.StringVar(&logFormat, "log-format", "", "specifies the log format: ltsv or json.")
	flag.BoolVar(&enablePprof, "enable-pprof", false, ".")
	flag.BoolVar(&showVersion, "v", false, "show version number.")
	flag.BoolVar(&showVersion, "version", false, "show version number.")
	flag.BoolVar(&gunfish.OutputHookStdout, "output-hook-stdout", false, "merge stdout of hook command to gunfish's stdout")
	flag.BoolVar(&gunfish.OutputHookStderr, "output-hook-stderr", false, "merge stderr of hook command to gunfish's stderr")

	flag.StringVar(&logLevel, "log-level", "info", "set the log level (debug, warn, info)")
	flag.Parse()

	if showVersion {
		fmt.Printf("Compiler: %s %s\n", runtime.Compiler, runtime.Version())
		fmt.Printf("Gunfish version: %s\n", version)
		return
	}

	initLogrus(logFormat, logLevel)

	c, err := config.LoadConfig(confPath)
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
		apns.ClientTransport = func(cert tls.Certificate) *http.Transport {
			return &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
					Certificates:       []tls.Certificate{cert},
				},
			}
		}
	default:
		logrus.Errorf("Unknown environment: %s. Please look at help.", environment)
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

func initLogrus(format string, logLevel string) {
	switch format {
	case "ltsv":
		logrus.SetFormatter(&gunfish.LtsvFormatter{})
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}

	lvl, err := logrus.ParseLevel(logLevel)
	if err != nil {
		lvl = logrus.InfoLevel
	}

	logrus.SetLevel(lvl)
}
