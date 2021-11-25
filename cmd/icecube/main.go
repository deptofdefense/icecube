// =================================================================
//
// Work of the U.S. Department of Defense, Defense Digital Service.
// Released as open source under the MIT License.  See LICENSE file.
//
// =================================================================

package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/deptofdefense/icecube/pkg/log"
	"github.com/deptofdefense/icecube/pkg/server"
)

const (
	IcecubeVersion = "1.0.0"
)

const (
	TLSVersion1_0 = "1.0"
	TLSVersion1_1 = "1.1"
	TLSVersion1_2 = "1.2"
	TLSVersion1_3 = "1.3"
)

var (
	SupportedTLSVersions = []string{
		TLSVersion1_0,
		TLSVersion1_1,
		TLSVersion1_2,
		TLSVersion1_3,
	}
	TLSVersionIdentifiers = map[string]uint16{
		TLSVersion1_0: tls.VersionTLS10,
		TLSVersion1_1: tls.VersionTLS11,
		TLSVersion1_2: tls.VersionTLS12,
		TLSVersion1_3: tls.VersionTLS13,
	}
)

const (
	CurveP256 = "CurveP256"
	CurveP384 = "CurveP384"
	CurveP521 = "CurveP521"
	X25519    = "X25519"
)

const (
	BehaviorRedirect = "redirect"
	BehaviorNone     = "none"
)

var (
	Behaviors = []string{
		BehaviorRedirect,
		BehaviorNone,
	}
)

var (
	DefaultCurveIDs = []string{
		X25519,
		CurveP256,
		CurveP384,
		CurveP521,
	}
	SupportedCurveIDs = []string{
		CurveP256,
		CurveP384,
		CurveP521,
		X25519,
	}
	TLSCurveIdentifiers = map[string]tls.CurveID{
		CurveP256: tls.CurveP256,
		CurveP384: tls.CurveP384,
		CurveP521: tls.CurveP521,
		X25519:    tls.X25519,
	}
)

func stringSliceContains(stringSlice []string, value string) bool {
	for _, x := range stringSlice {
		if value == x {
			return true
		}
	}
	return false
}

func stringSliceIndex(stringSlice []string, value string) int {
	for i, x := range stringSlice {
		if value == x {
			return i
		}
	}
	return -1
}

const (
	flagListenAddress   = "addr"
	flagRedirectAddress = "redirect"
	flagPublicLocation  = "public-location"
	//
	flagServerCert = "server-cert"
	flagServerKey  = "server-key"
	//
	flagRootPath = "root"
	//
	flagTimeoutRead  = "timeout-read"
	flagTimeoutWrite = "timeout-write"
	flagTimeoutIdle  = "timeout-idle"
	//
	flagTLSMinVersion       = "tls-min-version"
	flagTLSMaxVersion       = "tls-max-version"
	flagTLSCipherSuites     = "tls-cipher-suites"
	flagTLSCurvePreferences = "tls-curve-preferences"
	//
	flagBehaviorNotFound = "behavior-not-found"
	//
	flagLogPath    = "log"
	flagKeyLogPath = "keylog"
	//
	flagUnsafe = "unsafe"
	flagDryRun = "dry-run"
)

type File struct {
	ModTime string
	Size    int64
	Type    string
	Path    string
}

func initFlags(flag *pflag.FlagSet) {
	flag.String(flagPublicLocation, "", "the public location of the server used for redirects")
	flag.StringP(flagListenAddress, "a", ":8080", "address that icecube will listen on")
	flag.String(flagRedirectAddress, "", "address that icecube will listen to and redirect requests to the public location")
	flag.String(flagServerCert, "", "path to server public cert")
	flag.String(flagServerKey, "", "path to server private key")
	flag.StringP(flagRootPath, "r", "", "path to the document root served")
	flag.StringP(flagLogPath, "l", "-", "path to the log output.  Defaults to stdout.")
	flag.String(flagKeyLogPath, "", "path to the key log output.  Also requires unsafe flag.")
	flag.String(flagBehaviorNotFound, BehaviorNone, "default behavior when a file is not found.  One of: "+strings.Join(Behaviors, ","))
	initTimeoutFlags(flag)
	initTLSFlags(flag)
	flag.Bool(flagUnsafe, false, "allow unsafe configuration")
	flag.Bool(flagDryRun, false, "exit after checking configuration")
}

func initTimeoutFlags(flag *pflag.FlagSet) {
	flag.String(flagTimeoutRead, "15m", "maximum duration for reading the entire request")
	flag.String(flagTimeoutWrite, "5m", "maximum duration before timing out writes of the response")
	flag.String(flagTimeoutIdle, "5m", "maximum amount of time to wait for the next request when keep-alives are enabled")
}

func initTLSFlags(flag *pflag.FlagSet) {
	flag.String(flagTLSMinVersion, TLSVersion1_0, "minimum TLS version accepted for requests")
	flag.String(flagTLSMaxVersion, TLSVersion1_3, "maximum TLS version accepted for requests")
	flag.String(flagTLSCipherSuites, "", "list of supported cipher suites for TLS versions up to 1.2 (TLS 1.3 is not configurable)")
	flag.String(flagTLSCurvePreferences, strings.Join(DefaultCurveIDs, ","), "curve preferences")
}

func initViper(cmd *cobra.Command) (*viper.Viper, error) {
	v := viper.New()
	err := v.BindPFlags(cmd.Flags())
	if err != nil {
		return v, fmt.Errorf("error binding flag set to viper: %w", err)
	}
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv() // set environment variables to overwrite config
	return v, nil
}

func checkConfig(v *viper.Viper) error {
	addr := v.GetString(flagListenAddress)
	if len(addr) == 0 {
		return fmt.Errorf("listen address is missing")
	}
	redirectAddress := v.GetString(flagRedirectAddress)
	if len(redirectAddress) > 0 {
		publicLocation := v.GetString(flagPublicLocation)
		if len(publicLocation) == 0 {
			return fmt.Errorf("public location is required when redirecting")
		}
		if !strings.HasPrefix(publicLocation, "https://") {
			return fmt.Errorf("public location must start with \"https://\"")
		}
	}
	serverCert := v.GetString(flagServerCert)
	if len(serverCert) == 0 {
		return fmt.Errorf("server cert is missing")
	}
	serverKey := v.GetString(flagServerKey)
	if len(serverKey) == 0 {
		return fmt.Errorf("server key is missing")
	}
	rootPath := v.GetString(flagRootPath)
	if len(rootPath) == 0 {
		return fmt.Errorf("root path is missing")
	}
	logPath := v.GetString(flagLogPath)
	if len(logPath) == 0 {
		return fmt.Errorf("log path is missing")
	}
	timeoutRead := v.GetString(flagTimeoutRead)
	if len(timeoutRead) == 0 {
		return fmt.Errorf("read timeout is missing")
	}
	timeoutReadDuration, err := time.ParseDuration(timeoutRead)
	if err != nil {
		return fmt.Errorf("error parsing read timeout: %w", err)
	}
	if timeoutReadDuration < 5*time.Second || timeoutReadDuration > 30*time.Minute {
		return fmt.Errorf("invalid read timeout %q, must be greater than or equal to 5 seconds and less than or equal to 30 minutes", timeoutReadDuration)
	}
	timeoutWrite := v.GetString(flagTimeoutWrite)
	if len(timeoutWrite) == 0 {
		return fmt.Errorf("write timeout is missing")
	}
	timeoutWriteDuration, err := time.ParseDuration(timeoutWrite)
	if err != nil {
		return fmt.Errorf("error parsing write timeout: %w", err)
	}
	if timeoutWriteDuration < 5*time.Second || timeoutWriteDuration > 30*time.Minute {
		return fmt.Errorf("invalid write timeout %q, must be greater than or equal to 5 seconds and less than or equal to 30 minutes", timeoutWriteDuration)
	}
	timeoutIdle := v.GetString(flagTimeoutIdle)
	if len(timeoutIdle) == 0 {
		return fmt.Errorf("idle timeout is missing")
	}
	timeoutIdleDuration, err := time.ParseDuration(timeoutIdle)
	if err != nil {
		return fmt.Errorf("error parsing idle timeout: %w", err)
	}
	if timeoutIdleDuration < 5*time.Second || timeoutIdleDuration > 30*time.Minute {
		return fmt.Errorf("invalid idle timeout %q, must be greater than or equal to 5 seconds and less than or equal to 30 minutes", timeoutIdleDuration)
	}
	if err := checkTLSConfig(v); err != nil {
		return fmt.Errorf("error with TLS configuration: %w", err)
	}
	return nil
}

func checkTLSConfig(v *viper.Viper) error {
	minVersion := v.GetString(flagTLSMinVersion)
	minVersionIndex := stringSliceIndex(SupportedTLSVersions, minVersion)
	if minVersionIndex == -1 {
		return fmt.Errorf("invalid minimum TLS version %q", minVersion)
	}
	maxVersion := v.GetString(flagTLSMaxVersion)
	maxVersionIndex := stringSliceIndex(SupportedTLSVersions, maxVersion)
	if maxVersionIndex == -1 {
		return fmt.Errorf("invalid maximum TLS version %q", maxVersion)
	}
	if minVersionIndex > maxVersionIndex {
		return fmt.Errorf("invalid TLS versions, minium version %q is greater than maximum version %q", minVersion, maxVersion)
	}
	curvePreferencesString := v.GetString(flagTLSCurvePreferences)
	if len(curvePreferencesString) == 0 {
		return fmt.Errorf("TLS curve preferences are missing")
	}
	curvePreferences := strings.Split(curvePreferencesString, ",")
	for _, curveID := range curvePreferences {
		if !stringSliceContains(SupportedCurveIDs, curveID) {
			return fmt.Errorf("invalid curve preference %q", curveID)
		}
	}
	return nil
}

func newTraceID() string {
	traceID, err := uuid.NewV4()
	if err != nil {
		return ""
	}
	return traceID.String()
}

func initLogger(path string) (*log.SimpleLogger, error) {

	if path == "-" {
		return log.NewSimpleLogger(os.Stdout), nil
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("error opening log file %q: %w", path, err)
	}

	return log.NewSimpleLogger(f), nil
}

func initKeyLogger(path string, unsafe bool) (io.Writer, error) {

	// if path is not defined or unsafe is not set, then return nil
	if len(path) == 0 || !unsafe {
		return nil, nil
	}

	if path == "-" {
		return nil, errors.New("stdout is not supported for key log")
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("error opening key log file %q: %w", path, err)
	}

	return f, nil
}

func getTLSVersion(r *http.Request) string {
	for k, v := range TLSVersionIdentifiers {
		if v == r.TLS.Version {
			return k
		}
	}
	return ""
}

func initCipherSuites(cipherSuiteNamesString string, supportedCipherSuites []*tls.CipherSuite) ([]uint16, error) {
	if len(cipherSuiteNamesString) == 0 {
		return nil, nil
	}
	cipherSuiteNames := strings.SplitN(cipherSuiteNamesString, ",", -1)
	cipherSuitesMap := map[string]uint16{}
	for _, cipherSuite := range supportedCipherSuites {
		cipherSuitesMap[cipherSuite.Name] = cipherSuite.ID
	}
	cipherSuites := make([]uint16, 0, len(cipherSuiteNames))
	for _, cipherSuiteName := range cipherSuiteNames {
		cipherSuiteID, ok := cipherSuitesMap[cipherSuiteName]
		if !ok {
			return nil, fmt.Errorf("unknown TLS cipher suite %q", cipherSuiteName)
		}
		cipherSuites = append(cipherSuites, cipherSuiteID)
	}
	return cipherSuites, nil
}

func initTLSConfig(v *viper.Viper, serverKeyPair tls.Certificate, minVersion string, maxVersion string, cipherSuites []uint16, keyLogger io.Writer) *tls.Config {

	config := &tls.Config{
		Certificates: []tls.Certificate{serverKeyPair},
		MinVersion:   TLSVersionIdentifiers[minVersion],
		MaxVersion:   TLSVersionIdentifiers[maxVersion],
		KeyLogWriter: keyLogger,
	}

	if len(cipherSuites) > 0 {
		config.CipherSuites = cipherSuites
	}

	if tlsCurvePreferencesString := v.GetString(flagTLSCurvePreferences); len(tlsCurvePreferencesString) > 0 {
		curvePreferences := make([]tls.CurveID, 0)
		for _, str := range strings.Split(tlsCurvePreferencesString, ",") {
			curvePreferences = append(curvePreferences, TLSCurveIdentifiers[str])
		}
		config.CurvePreferences = curvePreferences
	}
	return config
}

func main() {

	rootCommand := &cobra.Command{
		Use:                   `icecube [flags]`,
		DisableFlagsInUseLine: true,
		Short:                 "icecube is a file server.",
	}

	defaultsCommand := &cobra.Command{
		Use:                   `defaults`,
		DisableFlagsInUseLine: true,
		Short:                 "show defaults",
		SilenceErrors:         true,
		SilenceUsage:          true,
	}

	showDefaultTLSCipherSuites := &cobra.Command{
		Use:                   `tls-cipher-suites`,
		DisableFlagsInUseLine: true,
		Short:                 "show default TLS cipher suites",
		SilenceErrors:         true,
		SilenceUsage:          true,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := initViper(cmd)
			if err != nil {
				return fmt.Errorf("error initializing viper: %w", err)
			}
			if len(args) > 0 {
				return cmd.Usage()
			}
			supportedCipherSuites := tls.CipherSuites()
			names := make([]string, 0, len(supportedCipherSuites))
			for _, cipherSuite := range supportedCipherSuites {
				names = append(names, cipherSuite.Name)
			}
			fmt.Println(strings.Join(names, "\n"))
			return nil
		},
	}

	showDefaultTLSCurvePreferences := &cobra.Command{
		Use:                   `tls-curve-preferences`,
		DisableFlagsInUseLine: true,
		Short:                 "show default TLS curve preferences",
		SilenceErrors:         true,
		SilenceUsage:          true,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := initViper(cmd)
			if err != nil {
				return fmt.Errorf("error initializing viper: %w", err)
			}
			if len(args) > 0 {
				return cmd.Usage()
			}
			fmt.Println(strings.Join(DefaultCurveIDs, "\n"))
			return nil
		},
	}

	defaultsCommand.AddCommand(showDefaultTLSCipherSuites, showDefaultTLSCurvePreferences)

	serveCommand := &cobra.Command{
		Use:                   `serve [flags]`,
		DisableFlagsInUseLine: true,
		Short:                 "start the icecube server",
		SilenceErrors:         true,
		SilenceUsage:          true,
		RunE: func(cmd *cobra.Command, args []string) error {
			v, err := initViper(cmd)
			if err != nil {
				return fmt.Errorf("error initializing viper: %w", err)
			}

			if len(args) > 1 {
				return cmd.Usage()
			}

			if errConfig := checkConfig(v); errConfig != nil {
				return errConfig
			}

			logger, err := initLogger(v.GetString(flagLogPath))
			if err != nil {
				return fmt.Errorf("error initializing logger: %w", err)
			}

			unsafe := v.GetBool(flagUnsafe)

			if unsafe {
				_ = logger.Log("Unsafe configuration allowed", map[string]interface{}{
					"unsafe": unsafe,
				})
			}

			keyLogger, err := initKeyLogger(v.GetString(flagKeyLogPath), unsafe)
			if err != nil {
				return fmt.Errorf("error initializing key logger: %w", err)
			}

			if keyLogger != nil {
				_ = logger.Log("Logging TLS keys", map[string]interface{}{
					"unsafe": unsafe,
					"path":   v.GetString(flagKeyLogPath),
				})
			}

			listenAddress := v.GetString(flagListenAddress)
			redirectAddress := v.GetString(flagRedirectAddress)
			publicLocation := v.GetString(flagPublicLocation)
			rootPath := v.GetString(flagRootPath)

			root := afero.NewBasePathFs(afero.NewReadOnlyFs(afero.NewOsFs()), rootPath)

			serverKeyPair, err := tls.LoadX509KeyPair(v.GetString(flagServerCert), v.GetString(flagServerKey))
			if err != nil {
				return fmt.Errorf("error loading server key pair: %w", err)
			}

			tlsMinVersion := v.GetString(flagTLSMinVersion)

			tlsMaxVersion := v.GetString(flagTLSMaxVersion)

			cipherSuiteNames := v.GetString(flagTLSCipherSuites)

			supportedCipherSuites := tls.CipherSuites()

			cipherSuites, err := initCipherSuites(cipherSuiteNames, supportedCipherSuites)
			if err != nil {
				return fmt.Errorf("error initializing cipher suites: %w", err)
			}

			tlsConfig := initTLSConfig(v, serverKeyPair, tlsMinVersion, tlsMaxVersion, cipherSuites, keyLogger)

			redirectNotFound := v.GetString(flagBehaviorNotFound) == BehaviorRedirect

			httpsServer := &http.Server{
				Addr:         listenAddress,
				IdleTimeout:  v.GetDuration(flagTimeoutIdle),
				ReadTimeout:  v.GetDuration(flagTimeoutRead),
				WriteTimeout: v.GetDuration(flagTimeoutWrite),
				TLSConfig:    tlsConfig,
				ErrorLog:     log.WrapStandardLogger(logger),
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					//
					icecubeTraceID := newTraceID()
					//
					_ = logger.Log("Request", map[string]interface{}{
						"url":              r.URL.String(),
						"source":           r.RemoteAddr,
						"referer":          r.Header.Get("referer"),
						"host":             r.Host,
						"method":           r.Method,
						"icecube_trace_id": icecubeTraceID,
						"tls_version":      getTLSVersion(r),
					})

					// Get path from URL
					p := server.TrimTrailingForwardSlash(server.CleanPath(r.URL.Path))

					// If path is not clean
					if !server.CheckPath(p) {
						_ = logger.Log("Invalid path", map[string]interface{}{
							"icecube_trace_id": icecubeTraceID,
							"url":              r.URL.String(),
							"path":             p,
						})
						http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
						return
					}

					fi, err := root.Stat(p)
					if err != nil {
						if os.IsNotExist(err) {
							_ = logger.Log("Not found", map[string]interface{}{
								"path":             p,
								"icecube_trace_id": icecubeTraceID,
							})
							if redirectNotFound {
								http.Redirect(w, r, publicLocation, http.StatusSeeOther)
								return
							}
							http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
							return
						}
						_ = logger.Log("Error stating file", map[string]interface{}{
							"path":             p,
							"icecube_trace_id": icecubeTraceID,
						})
						http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
						return
					}
					if fi.IsDir() {
						indexPath := filepath.Join(p, "index.html")
						indexFileInfo, err := root.Stat(indexPath)
						if err != nil && !os.IsNotExist(err) {
							_ = logger.Log("Error stating index file", map[string]interface{}{
								"path":             indexPath,
								"icecube_trace_id": icecubeTraceID,
							})
							http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
							return
						}
						if os.IsNotExist(err) || indexFileInfo.IsDir() {
							// if index file does not exist or is a directory then return not found.
							http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
							return
						}
						server.ServeFile(w, r, root, indexPath, time.Time{}, false, nil)
						return
					}
					server.ServeFile(w, r, root, p, fi.ModTime(), true, nil)
				}),
			}
			// If dry run, then return before starting servers.
			if v.GetBool(flagDryRun) {
				return nil
			}
			//
			if len(redirectAddress) > 0 && len(publicLocation) > 0 {
				httpServer := &http.Server{
					Addr:     redirectAddress,
					ErrorLog: log.WrapStandardLogger(logger),
					Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						_ = logger.Log("Redirecting request", map[string]interface{}{
							"icecube_trace_id": newTraceID(),
							"url":              r.URL.String(),
							"target":           publicLocation,
						})
						http.Redirect(w, r, publicLocation, http.StatusSeeOther)
					}),
				}
				_ = logger.Log("Redirecting http to https", map[string]interface{}{
					"source": redirectAddress,
					"target": publicLocation,
				})
				go func() { _ = httpServer.ListenAndServe() }()
			}
			//
			_ = logger.Log("Starting server", map[string]interface{}{
				"addr":          listenAddress,
				"idleTimeout":   httpsServer.IdleTimeout.String(),
				"readTimeout":   httpsServer.ReadTimeout.String(),
				"writeTimeout":  httpsServer.WriteTimeout.String(),
				"tlsMinVersion": tlsMinVersion,
				"tlsMaxVersion": tlsMaxVersion,
			})
			return httpsServer.ListenAndServeTLS("", "")
		},
	}
	initFlags(serveCommand.Flags())

	versionCommand := &cobra.Command{
		Use:                   `version`,
		DisableFlagsInUseLine: true,
		Short:                 "show version",
		SilenceErrors:         true,
		SilenceUsage:          true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(IcecubeVersion)
			return nil
		},
	}

	rootCommand.AddCommand(defaultsCommand, serveCommand, versionCommand)

	if err := rootCommand.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "icecube: "+err.Error())
		_, _ = fmt.Fprintln(os.Stderr, "Try icecube --help for more information.")
		os.Exit(1)
	}
}
