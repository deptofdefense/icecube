// =================================================================
//
// Work of the U.S. Department of Defense, Defense Digital Service.
// Released as open source under the MIT License.  See LICENSE file.
//
// =================================================================

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/deptofdefense/icecube/pkg/fs"
	"github.com/deptofdefense/icecube/pkg/log"
	"github.com/deptofdefense/icecube/pkg/server"
	"github.com/deptofdefense/icecube/pkg/template"
)

const (
	IcecubeVersion = "1.0.1"
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
	NotFoundBehaviors = []string{
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

func unmarshalServerKeyPairs(str string) ([]tls.Certificate, error) {
	if len(str) == 0 {
		return []tls.Certificate{}, nil
	}
	serverKeyPairs := [][2]string{}
	err := json.Unmarshal([]byte(str), &serverKeyPairs)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling server key pairs: %w", err)
	}
	certificates := []tls.Certificate{}
	for i, kp := range serverKeyPairs {
		cert, err := tls.LoadX509KeyPair(kp[0], kp[1])
		if err != nil {
			return nil, fmt.Errorf("error loading server key pair %d: %w", i, err)
		}
		certificates = append(certificates, cert)
	}
	return certificates, nil
}

const (
	flagListenAddress   = "addr"
	flagRedirectAddress = "redirect"
	flagPublicLocation  = "public-location"
	//
	flagDefaultServerCert = "server-cert"
	flagDefaultServerKey  = "server-key"
	flagServerKeyPairs    = "server-key-pairs"
	//
	flagRootPath    = "root"
	flagFileSystems = "file-systems"
	//
	flagSites = "sites"
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
	flagDirectoryIndex         = "directory-index"
	flagDirectoryTemplate      = "directory-template"
	flagDirectoryTrailingSlash = "directory-trailing-slash"
	//
	flagMaxDirectoryEntries = "max-directory-entries"
	//
	flagLogPath    = "log"
	flagLogPerm    = "log-perm"
	flagKeyLogPath = "keylog"
	//
	flagUnsafe = "unsafe"
	flagDryRun = "dry-run"
	//
	flagAWSPartition          = "aws-partition"
	flagAWSProfile            = "aws-profile"
	flagAWSDefaultRegion      = "aws-default-region"
	flagAWSRegion             = "aws-region"
	flagAWSAccessKeyID        = "aws-access-key-id"
	flagAWSSecretAccessKey    = "aws-secret-access-key"
	flagAWSSessionToken       = "aws-session-token"
	flagAWSInsecureSkipVerify = "aws-insecure-skip-verify"
	flagAWSS3Endpoint         = "aws-s3-endpoint"
	flagAWSS3UsePathStyle     = "aws-s3-use-path-style"
)

type File struct {
	ModTime string
	Size    int64
	Type    string
	Path    string
}

func initServeFlags(flag *pflag.FlagSet) {
	flag.String(flagPublicLocation, "", "the public location of the server used for redirects")
	flag.StringP(flagListenAddress, "a", ":8080", "address that icecube will listen on")
	flag.String(flagRedirectAddress, "", "address that icecube will listen to and redirect requests to the public location")
	flag.String(flagDefaultServerCert, "", "path to default server public cert")
	flag.String(flagDefaultServerKey, "", "path to default server private key")
	flag.String(flagServerKeyPairs, "", "additional server key pairs in the format of a json array of arrays [[path to server public cert, path to server private key],...]")
	flag.StringP(flagRootPath, "r", "", "path to the default document root served")
	flag.String(flagFileSystems, "", "additional file systems in the format of a json array of strings")
	flag.String(flagSites, "", "sites hosted by the server in the format of a json map of server name to file system")
	flag.StringP(flagLogPath, "l", "-", "path to the log output.  Defaults to stdout.")
	flag.String(flagLogPerm, "0600", "file permissions for log output file as unix file mode.")
	flag.String(flagKeyLogPath, "", "path to the key log output.  Also requires unsafe flag.")
	flag.String(flagDirectoryIndex, "", "index file for directories")
	flag.String(flagDirectoryTemplate, "", "path to directory template")
	flag.Bool(flagDirectoryTrailingSlash, false, "append trailing slash to directories")
	flag.Int(flagMaxDirectoryEntries, -1, "maximum directory entries returned")
	flag.String(flagBehaviorNotFound, BehaviorNone, "default behavior when a file is not found.  One of: "+strings.Join(NotFoundBehaviors, ","))
	initTimeoutFlags(flag)
	initTLSFlags(flag)
	flag.Bool(flagUnsafe, false, "allow unsafe configuration")
	flag.Bool(flagDryRun, false, "exit after checking configuration")
	initAWSFlags(flag)
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

func initAWSFlags(flag *pflag.FlagSet) {
	flag.String(flagAWSPartition, "", "AWS Partition")
	flag.String(flagAWSProfile, "", "AWS Profile")
	flag.String(flagAWSDefaultRegion, "", "AWS Default Region")
	flag.String(flagAWSRegion, "", "AWS Region (overrides default region)")
	flag.String(flagAWSAccessKeyID, "", "AWS Access Key ID")
	flag.String(flagAWSSecretAccessKey, "", "AWS Secret Access Key")
	flag.String(flagAWSSessionToken, "", "AWS Session Token")
	flag.Bool(flagAWSInsecureSkipVerify, false, "Skip verification of AWS TLS certificate")
	flag.String(flagAWSS3Endpoint, "", "AWS S3 Endpoint URL")
	flag.Bool(flagAWSS3UsePathStyle, false, "Use path-style addressing (default is to use virtual-host-style addressing)")
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

func initS3Client(v *viper.Viper) *s3.Client {
	accessKeyID := v.GetString(flagAWSAccessKeyID)
	secretAccessKey := v.GetString(flagAWSSecretAccessKey)
	sessionToken := v.GetString(flagAWSSessionToken)
	usePathStyle := v.GetBool(flagAWSS3UsePathStyle)

	region := v.GetString(flagAWSRegion)
	if len(region) == 0 {
		if defaultRegion := v.GetString(flagAWSDefaultRegion); len(defaultRegion) > 0 {
			region = defaultRegion
		}
	}

	config := aws.Config{
		RetryMaxAttempts: 3,
		Region:           region,
	}

	partition := v.GetString(flagAWSPartition)
	if len(partition) == 0 {
		partition = "aws"
	}

	if e := v.GetString(flagAWSS3Endpoint); len(e) > 0 {
		config.EndpointResolverWithOptions = aws.EndpointResolverWithOptionsFunc(func(service string, region string, options ...interface{}) (aws.Endpoint, error) {
			if service == s3.ServiceID {
				endpoint := aws.Endpoint{
					PartitionID:   partition,
					URL:           e,
					SigningRegion: region,
				}
				return endpoint, nil
			}
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		})
	}

	if len(accessKeyID) > 0 && len(secretAccessKey) > 0 {
		config.Credentials = credentials.NewStaticCredentialsProvider(
			accessKeyID,
			secretAccessKey,
			sessionToken)
	}

	insecureSkipVerify := v.GetBool(flagAWSInsecureSkipVerify)
	if insecureSkipVerify {
		config.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}

	return s3.NewFromConfig(config, func(o *s3.Options) {
		o.UsePathStyle = usePathStyle
	})
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
	serverKeyPairs := v.GetString(flagServerKeyPairs)
	if len(serverKeyPairs) > 0 {
		if err := json.Unmarshal([]byte(serverKeyPairs), &([][2]string{})); err != nil {
			return fmt.Errorf("invalid format for server key pairs %s: %w", serverKeyPairs, err)
		}
	} else {
		defaultServerCert := v.GetString(flagDefaultServerCert)
		if len(defaultServerCert) == 0 {
			return fmt.Errorf("default server cert is missing")
		}
		defaultServerKey := v.GetString(flagDefaultServerKey)
		if len(defaultServerKey) == 0 {
			return fmt.Errorf("default server key is missing")
		}
	}
	fileSystems := v.GetString(flagFileSystems)
	if len(fileSystems) > 0 {
		if err := json.Unmarshal([]byte(fileSystems), &([]string{})); err != nil {
			return fmt.Errorf("invalid format for file systems: %w", err)
		}
	} else {
		rootPath := v.GetString(flagRootPath)
		if len(rootPath) == 0 {
			return fmt.Errorf("root path is missing")
		}
	}

	sites := v.GetString(flagSites)
	if len(sites) > 0 {
		if err := json.Unmarshal([]byte(sites), &(map[string]string{})); err != nil {
			return fmt.Errorf("invalid format for sites: %w", err)
		}
	}

	logPath := v.GetString(flagLogPath)
	if len(logPath) == 0 {
		return fmt.Errorf("log path is missing")
	}
	logPerm := v.GetString(flagLogPerm)
	if len(logPerm) == 0 {
		return fmt.Errorf("log perm is missing")
	}
	_, err := strconv.ParseUint(logPerm, 8, 32)
	if err != nil {
		return fmt.Errorf("invalid format for log perm: %s", logPerm)
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

func initLogger(path string, perm string) (*log.SimpleLogger, error) {

	if path == "-" {
		return log.NewSimpleLogger(os.Stdout), nil
	}

	fileMode := os.FileMode(0600)

	if len(perm) > 0 {
		fm, err := strconv.ParseUint(perm, 8, 32)
		if err != nil {
			return nil, fmt.Errorf("error parsing file permissions for log file from %q", perm)
		}
		fileMode = os.FileMode(fm)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, fileMode)
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

func getNamesForCertificate(cert x509.Certificate) []string {
	names := []string{}
	if cert.Subject.CommonName != "" && len(cert.DNSNames) == 0 {
		names = append(names, cert.Subject.CommonName)
	}
	for _, san := range cert.DNSNames {
		names = append(names, san)
	}
	return names
}

func getLeafForCertficate(cert tls.Certificate) (*x509.Certificate, error) {
	if cert.Leaf != nil {
		return cert.Leaf, nil
	}
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("error parsing leaf certificate for certificate: %w", err)
	}
	return leaf, nil
}

func buildNameToCertificate(defaultCertificate *tls.Certificate, certificates []tls.Certificate) (map[string]*tls.Certificate, error) {
	// copied from crypto/tls/common.go in the Go Standard Library
	nameToCertificate := map[string]*tls.Certificate{}
	// Append default certificate to end of certificates
	if defaultCertificate != nil {
		certificates = append([]tls.Certificate{*defaultCertificate}, certificates...)
	}
	for i, _ := range certificates {
		c := certificates[len(certificates)-1-i]
		leaf, err := getLeafForCertficate(c)
		if err != nil {
			return nil, fmt.Errorf("error getting leaf for certificate %d: %w", i, err)
		}
		names := getNamesForCertificate(*leaf)
		for _, name := range names {
			nameToCertificate[name] = &c
		}
	}
	return nameToCertificate, nil
}

func initTLSConfig(v *viper.Viper, defaultCertificate *tls.Certificate, certificates []tls.Certificate, minVersion string, maxVersion string, cipherSuites []uint16, keyLogger io.Writer) (*tls.Config, error) {

	config := &tls.Config{
		MinVersion:   TLSVersionIdentifiers[minVersion],
		MaxVersion:   TLSVersionIdentifiers[maxVersion],
		KeyLogWriter: keyLogger,
	}
	if len(certificates) > 0 {
		nameToCertificate, err := buildNameToCertificate(defaultCertificate, certificates)
		if err != nil {
			return nil, fmt.Errorf("error building name to certificate map: %w", err)
		}
		config.GetCertificate = func(clientHelloInfo *tls.ClientHelloInfo) (*tls.Certificate, error) {
			if len(clientHelloInfo.ServerName) == 0 {
				if defaultCertificate != nil {
					return defaultCertificate, nil
				}
				return &certificates[0], nil
			}
			if c, ok := nameToCertificate[clientHelloInfo.ServerName]; ok {
				return c, nil
			}
			if defaultCertificate != nil {
				return defaultCertificate, nil
			}
			return &certificates[0], nil
		}
	} else {
		config.Certificates = []tls.Certificate{*defaultCertificate}
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
	return config, nil
}

func initFileSystem(ctx context.Context, rootPath string, s3Client *s3.Client, maxDirectoryEntries int) fs.FileSystem {
	if strings.HasPrefix(rootPath, "s3://") {
		rootParts := strings.Split(rootPath[len("s3://"):], "/")
		bucket := rootParts[0]
		prefix := strings.Join(rootParts[1:], "/")
		bucketCreationDate := time.Now()
		listBucketsOutput, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
		if err == nil {
			for _, b := range listBucketsOutput.Buckets {
				if bucket == aws.ToString(b.Name) {
					bucketCreationDate = aws.ToTime(b.CreationDate)
					break
				}
			}
		}
		return fs.NewS3FileSystem(bucket, prefix, s3Client, bucketCreationDate, maxDirectoryEntries)
	}

	return fs.NewLocalFileSystem(rootPath)
}

func initFileSystems(ctx context.Context, v *viper.Viper, maxDirectoryEntries int) (map[string]fs.FileSystem, error) {
	rootPath := v.GetString(flagRootPath)
	fileSystemPathsString := v.GetString(flagFileSystems)
	fileSystemPathsSlice := []string{}
	if len(fileSystemPathsString) > 0 {
		err := json.Unmarshal([]byte(fileSystemPathsString), &fileSystemPathsSlice)
		if err != nil {
			return nil, fmt.Errorf("invalid format for file systems: %w", err)
		}
	}

	s3ClientNeeded := false
	if strings.HasPrefix(rootPath, "s3://") {
		s3ClientNeeded = true
	} else {
		for _, str := range fileSystemPathsSlice {
			if strings.HasPrefix(str, "s3://") {
				s3ClientNeeded = true
				break
			}
		}
	}

	var s3Client *s3.Client

	if s3ClientNeeded {
		s3Client = initS3Client(v)
	}

	fileSystems := map[string]fs.FileSystem{}

	if len(rootPath) > 0 {
		fileSystems[rootPath] = initFileSystem(ctx, rootPath, s3Client, maxDirectoryEntries)
	}

	if len(fileSystemPathsSlice) > 0 {
		for _, fileSystemPath := range fileSystemPathsSlice {
			fileSystems[fileSystemPath] = initFileSystem(ctx, fileSystemPath, s3Client, maxDirectoryEntries)
		}
	}

	return fileSystems, nil
}

func initSites(v *viper.Viper) (map[string]string, error) {
	sitesString := v.GetString(flagSites)
	sitesMap := map[string]string{}
	if len(sitesString) > 0 {
		err := json.Unmarshal([]byte(sitesString), &sitesMap)
		if err != nil {
			return nil, fmt.Errorf("invalid format for sites: %w", err)
		}
	}
	return sitesMap, nil
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
		Example: `serve --addr :8080 --server-cert server.crt --server-key server.key --root /www
serve --addr :8080 --server-key-pairs '[["server.crt", "server.key"]]' --file-systems ["/www"] --sites '{"localhost": "/www"}'`,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {

			ctx := cmd.Context()

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

			logger, err := initLogger(v.GetString(flagLogPath), v.GetString(flagLogPerm))
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

			defaultRootPath := v.GetString(flagRootPath)

			maxDirectoryEntries := v.GetInt(flagMaxDirectoryEntries)
			fileSystems, err := initFileSystems(ctx, v, maxDirectoryEntries)
			if err != nil {
				return fmt.Errorf("error initializing file systems: %w", err)
			}

			sites, err := initSites(v)
			if err != nil {
				return fmt.Errorf("error initializing sites: %w", err)
			}

			var defaultServerKeyPair *tls.Certificate
			if len(v.GetString(flagDefaultServerCert)) > 0 {
				kp, loadX509KeyPairError := tls.LoadX509KeyPair(v.GetString(flagDefaultServerCert), v.GetString(flagDefaultServerKey))
				if loadX509KeyPairError != nil {
					return fmt.Errorf("error loading default server key pair: %w", loadX509KeyPairError)
				}
				defaultServerKeyPair = &kp
			}

			serverKeyPairs, err := unmarshalServerKeyPairs(v.GetString((flagServerKeyPairs)))
			if err != nil {
				return fmt.Errorf("error loading server key pairs: %w", err)
			}

			if defaultServerKeyPair == nil {
				// if default server key pair is nil, then set the value to the first key pair provided
				defaultServerKeyPair = &serverKeyPairs[0]
			} else {
				// if default server key pair is not nil, then add to the slice of key pairs
				serverKeyPairs = append(serverKeyPairs, *defaultServerKeyPair)
			}

			tlsMinVersion := v.GetString(flagTLSMinVersion)

			tlsMaxVersion := v.GetString(flagTLSMaxVersion)

			cipherSuiteNames := v.GetString(flagTLSCipherSuites)

			supportedCipherSuites := tls.CipherSuites()

			cipherSuites, err := initCipherSuites(cipherSuiteNames, supportedCipherSuites)
			if err != nil {
				return fmt.Errorf("error initializing cipher suites: %w", err)
			}

			tlsConfig, err := initTLSConfig(v, defaultServerKeyPair, serverKeyPairs, tlsMinVersion, tlsMaxVersion, cipherSuites, keyLogger)
			if err != nil {
				return fmt.Errorf("error initializing TLS config: %w", err)
			}

			redirectNotFound := v.GetString(flagBehaviorNotFound) == BehaviorRedirect

			directoryIndex := v.GetString(flagDirectoryIndex)
			directoryTemplatePath := v.GetString(flagDirectoryTemplate)
			directoryTrailingSlash := v.GetBool(flagDirectoryTrailingSlash)

			var directoryTemplate template.Template
			if len(directoryTemplatePath) > 0 {
				t, err := template.ParseFile("index.html", directoryTemplatePath)
				if err != nil {
					return fmt.Errorf("error parsing directory template: %w", err)
				}
				directoryTemplate = t
				_ = logger.Log("Using directory template", map[string]interface{}{
					"path": directoryTemplatePath,
				})
			}

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
					tlsServerName := r.TLS.ServerName
					requestContext := r.Context()
					//
					_ = logger.Log("Request", map[string]interface{}{
						"url":              r.URL.String(),
						"source":           r.RemoteAddr,
						"referer":          r.Header.Get("referer"),
						"host":             r.Host,
						"method":           r.Method,
						"icecube_trace_id": icecubeTraceID,
						"tls_version":      getTLSVersion(r),
						"tls_server_name":  tlsServerName,
					})

					// Check site
					fileSystemPath := defaultRootPath
					if len(sites) > 0 {
						str, ok := sites[tlsServerName]
						if !ok {
							_ = logger.Log("Could not find site for server name", map[string]interface{}{
								"icecube_trace_id": icecubeTraceID,
								"url":              r.URL.String(),
								"host":             r.Host,
								"tls_server_name":  tlsServerName,
							})
							http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
							return
						}
						fileSystemPath = str
					}

					fs := fileSystems[fileSystemPath]

					// Get path from URL
					cleanPath := server.CleanPath(r.URL.Path)
					trimmedPath := server.TrimTrailingForwardSlash(cleanPath)

					// If path is not clean
					if !server.CheckPath(trimmedPath) {
						_ = logger.Log("Invalid path", map[string]interface{}{
							"icecube_trace_id": icecubeTraceID,
							"url":              r.URL.String(),
							"path":             trimmedPath,
						})
						http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
						return
					}

					fi, err := fs.Stat(requestContext, trimmedPath)
					if err != nil {
						if fs.IsNotExist(err) {
							_ = logger.Log("Not found", map[string]interface{}{
								"path":             trimmedPath,
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
							"path":             trimmedPath,
							"icecube_trace_id": icecubeTraceID,
							"error":            err.Error(),
						})
						http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
						return
					}
					if fi.IsDir() {
						if directoryTrailingSlash {
							if !strings.HasSuffix(cleanPath, "/") {
								http.Redirect(w, r, cleanPath+"/", http.StatusSeeOther)
								return
							}
						}
						if len(directoryIndex) == 0 {
							if directoryTemplate == nil {
								// if no directory index or directory template is to be checked, then return not found
								http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
								return
							}
							directoryEntries, readDirError := fs.ReadDir(requestContext, trimmedPath)
							if readDirError != nil {
								if fs.IsNotExist(readDirError) && trimmedPath == "/" {
									_ = logger.Log("Root directory not found", map[string]interface{}{
										"path":             trimmedPath,
										"icecube_trace_id": icecubeTraceID,
									})
									http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
									return
								}
								_ = logger.Log("Error reading directory", map[string]interface{}{
									"path":             trimmedPath,
									"icecube_trace_id": icecubeTraceID,
									"error":            readDirError.Error(),
								})
								http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
								return
							}
							buf := bytes.NewBuffer([]byte{})
							// Render Directory Template
							_ = logger.Log("Rendering directory template", map[string]interface{}{
								"path":              trimmedPath,
								"icecube_trace_id":  icecubeTraceID,
								"directory_entries": len(directoryEntries),
							})
							executeError := directoryTemplate.Execute(buf, map[string]interface{}{
								"Name":             trimmedPath,
								"DirectoryEntries": directoryEntries,
								"IcecubeVersion":   IcecubeVersion,
							})
							if executeError != nil {
								_ = logger.Log("Error rendering directory template", map[string]interface{}{
									"path":             trimmedPath,
									"icecube_trace_id": icecubeTraceID,
									"error":            executeError.Error(),
								})
								http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
								return
							}
							// Serve Rendered Directory Template
							server.ServeContent(w, r, trimmedPath, bytes.NewReader(buf.Bytes()), fi.ModTime(), false, nil)
							return
						}

						indexPath := ""
						// if directory index starts with / then look for it in the root directory of the server.
						if strings.HasPrefix(directoryIndex, "/") {
							indexPath = directoryIndex
						} else {
							indexPath = fs.Join(trimmedPath, directoryIndex)
						}

						indexFileInfo, err := fs.Stat(requestContext, indexPath)
						if err != nil && !fs.IsNotExist(err) {
							_ = logger.Log("Error stating index file", map[string]interface{}{
								"path":             indexPath,
								"icecube_trace_id": icecubeTraceID,
							})
							http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
							return
						}
						if fs.IsNotExist(err) || indexFileInfo.IsDir() {
							// if index file does not exist or is a directory then return not found.
							http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
							return
						}
						server.ServeFile(w, r, fs, indexPath, time.Time{}, false, nil)
						return
					}
					server.ServeFile(w, r, fs, trimmedPath, fi.ModTime(), true, nil)
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
	initServeFlags(serveCommand.Flags())

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
