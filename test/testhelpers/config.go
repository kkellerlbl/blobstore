package testhelpers

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/ini.v1"
)

const (
	// TestConfigEnv is the key for an environment variable containing the path to the test
	// configuration file.
	TestConfigEnv = "BLOBSTORE_TEST_CFG"
	// TestSection is the section of the test.cfg ini file where the tests look for configuration
	// values
	TestSection = "BlobstoreTest"
	// TestMinioExe is the key in the config file for the minio executable path.
	TestMinioExe = "test.minio.exe"
	// TestMongoExe is the key in the config file for the mongo executable path.
	TestMongoExe = "test.mongo.exe"
	// TestUseWiredTiger denotes that the MongoDB WiredTiger storage engine should be used.
	TestUseWiredTiger = "test.mongo.wired_tiger"
	// TestJarsDir is the key in the config file for the path to the KBase jars directory.
	TestJarsDir = "test.jars.dir"
	// TestTempDir is the key in the config file for the temporary directory.
	TestTempDir = "test.temp.dir"
	// TestDeleteTempDir is the key in the config file for whether the temporary directory
	// should be deleted when the tests are complete. Any value other than 'false' is treated
	// as true
	TestDeleteTempDir = "test.delete.temp.dir"
)

// TestConfig contains the test configuration.
type TestConfig struct {
	MinioExePath  string
	MongoExePath  string
	UseWiredTiger bool
	JarsDir       string
	TempDir       string
	DeleteTempDir bool
}

// GetConfig provides the test configuration.
// It expects the path to the test config file to be provided in the TestConfigEnv environment
// variable.
func GetConfig() (*TestConfig, error) {
	configfile := os.Getenv(TestConfigEnv)
	if configfile == "" {
		return nil, fmt.Errorf(
			"Must supply absolute path to test config file in %v environment variable",
			TestConfigEnv)
	}
	ini, err := ini.Load(configfile)
	if err != nil {
		return nil, err
	}
	sec, err := ini.GetSection(TestSection)
	if err != nil {
		return nil, err
	}
	minio, err := getValue(sec, TestMinioExe, configfile, true)
	if err != nil {
		return nil, err
	}
	mongo, err := getValue(sec, TestMongoExe, configfile, true)
	if err != nil {
		return nil, err
	}
	wiredTiger, err := getValue(sec, TestUseWiredTiger, configfile, false)
	if err != nil {
		return nil, err
	}
	jarsdir, err := getValue(sec, TestJarsDir, configfile, true)
	if err != nil {
		return nil, err
	}
	tempDir, err := getValue(sec, TestTempDir, configfile, true)
	if err != nil {
		return nil, err
	}
	del, err := getValue(sec, TestDeleteTempDir, configfile, false)
	if err != nil {
		return nil, err
	}

	return &TestConfig{
			MinioExePath:  minio,
			MongoExePath:  mongo,
			UseWiredTiger: wiredTiger == "true",
			JarsDir:       jarsdir,
			TempDir:       tempDir,
			DeleteTempDir: del != "false",
		},
		nil
}

func getValue(s *ini.Section, key string, file string, required bool) (string, error) {
	val := strings.TrimSpace(s.Key(key).String())
	if val == "" && required {
		return "", fmt.Errorf("Required key %v in section %v in config file %v is missing a value",
			key, TestSection, file)
	}
	return val, nil

}
