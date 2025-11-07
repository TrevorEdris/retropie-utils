package integration

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/TrevorEdris/retropie-utils/pkg/fs"
	"github.com/TrevorEdris/retropie-utils/pkg/storage"
	"github.com/TrevorEdris/retropie-utils/tools/syncer/pkg/syncer"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testBucketName = "test-retropie-bucket"
	testUsername   = "testuser"
)

type testEnvironment struct {
	localstackContainer testcontainers.Container
	endpoint            string
	tempRomsDir         string
	configFile          string
}

func setupTestEnvironment() (*testEnvironment, error) {
	ctx := context.Background()

	// Create LocalStack container
	req := testcontainers.ContainerRequest{
		Image:        "localstack/localstack:latest",
		ExposedPorts: []string{"4566/tcp"},
		Env: map[string]string{
			"SERVICES":           "s3",
			"DEFAULT_REGION":     "us-east-1",
			"AWS_DEFAULT_REGION": "us-east-1",
		},
		WaitingFor: wait.ForHTTP("/_localstack/health").
			WithPort("4566/tcp").
			WithStatusCodeMatcher(func(status int) bool {
				return status == http.StatusOK
			}).
			WithStartupTimeout(60 * time.Second),
	}

	localstackC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start LocalStack container: %w", err)
	}

	// Get endpoint
	port, err := localstackC.MappedPort(ctx, "4566")
	if err != nil {
		localstackC.Terminate(ctx)
		return nil, fmt.Errorf("failed to get mapped port: %w", err)
	}

	host, err := localstackC.Host(ctx)
	if err != nil {
		localstackC.Terminate(ctx)
		return nil, fmt.Errorf("failed to get host: %w", err)
	}

	endpoint := fmt.Sprintf("http://%s:%s", host, port.Port())

	// Set AWS environment variables
	os.Setenv("AWS_ENDPOINT", endpoint)
	os.Setenv("AWS_ACCESS_KEY_ID", "test")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	os.Setenv("AWS_REGION", "us-east-1")

	// Create temporary directory for ROMs
	tempRomsDir, err := os.MkdirTemp("", "syncer-integration-roms-*")
	if err != nil {
		localstackC.Terminate(ctx)
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Create test ROMs directory structure
	gbDir := filepath.Join(tempRomsDir, "gb")
	gbaDir := filepath.Join(tempRomsDir, "gba")
	gbcDir := filepath.Join(tempRomsDir, "gbc")
	if err := os.MkdirAll(gbDir, 0755); err != nil {
		localstackC.Terminate(ctx)
		return nil, fmt.Errorf("failed to create gb dir: %w", err)
	}
	if err := os.MkdirAll(gbaDir, 0755); err != nil {
		localstackC.Terminate(ctx)
		return nil, fmt.Errorf("failed to create gba dir: %w", err)
	}
	if err := os.MkdirAll(gbcDir, 0755); err != nil {
		localstackC.Terminate(ctx)
		return nil, fmt.Errorf("failed to create gbc dir: %w", err)
	}

	// Create test files
	testFiles := []struct {
		path    string
		content string
	}{
		{filepath.Join(gbDir, "test-rom.gb"), "fake gb rom content"},
		{filepath.Join(gbDir, "test-save.srm"), "fake save content"},
		{filepath.Join(gbDir, "test-state.state"), "fake state content"},
		{filepath.Join(gbaDir, "test-gba-rom.gba"), "fake gba rom content"},
		{filepath.Join(gbaDir, "test-gba-save.sav"), "fake gba save content"},
		{filepath.Join(gbcDir, "test-gbc-rom.gbc"), "fake gbc rom content"},
	}

	for _, tf := range testFiles {
		if err := os.WriteFile(tf.path, []byte(tf.content), 0644); err != nil {
			localstackC.Terminate(ctx)
			return nil, fmt.Errorf("failed to create test file %s: %w", tf.path, err)
		}
	}

	// Create config file
	configFile, err := createTestConfigFile(tempRomsDir, endpoint)
	if err != nil {
		localstackC.Terminate(ctx)
		return nil, fmt.Errorf("failed to create config file: %w", err)
	}

	return &testEnvironment{
		localstackContainer: localstackC,
		endpoint:            endpoint,
		tempRomsDir:         tempRomsDir,
		configFile:          configFile,
	}, nil
}

func teardownTestEnvironment(env *testEnvironment) error {
	ctx := context.Background()

	// Clean up environment variables
	os.Unsetenv("AWS_ENDPOINT")
	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	os.Unsetenv("AWS_REGION")

	// Remove temp directory
	if env.tempRomsDir != "" {
		os.RemoveAll(env.tempRomsDir)
	}

	// Remove config file
	if env.configFile != "" {
		os.Remove(env.configFile)
	}

	// Terminate container
	if env.localstackContainer != nil {
		return env.localstackContainer.Terminate(ctx)
	}

	return nil
}

func createTestConfigFile(romsDir, endpoint string) (string, error) {
	configContent := fmt.Sprintf(`username: %s
romsFolder: %s
storage:
  s3:
    enabled: true
    bucket: %s
    createMissingResources: true
sync:
  roms: true
  saves: true
  states: true
`, testUsername, romsDir, testBucketName)

	tmpFile, err := os.CreateTemp("", "syncer-config-*.yaml")
	if err != nil {
		return "", err
	}

	if _, err := tmpFile.WriteString(configContent); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", err
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

func createS3Bucket(ctx context.Context, endpoint string) error {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{
						URL:           endpoint,
						SigningRegion: region,
						Source:        aws.EndpointSourceCustom,
					}, nil
				},
			),
		),
	)
	if err != nil {
		return err
	}

	client := awss3.NewFromConfig(cfg, func(o *awss3.Options) {
		o.UsePathStyle = true
	})

	_, err = client.CreateBucket(ctx, &awss3.CreateBucketInput{
		Bucket: aws.String(testBucketName),
	})
	return err
}

func listS3Objects(ctx context.Context, endpoint string) ([]string, error) {
	return listS3ObjectsWithBucket(ctx, endpoint, testBucketName)
}

func listS3ObjectsWithBucket(ctx context.Context, endpoint, bucketName string) ([]string, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{
						URL:           endpoint,
						SigningRegion: region,
						Source:        aws.EndpointSourceCustom,
					}, nil
				},
			),
		),
	)
	if err != nil {
		return nil, err
	}

	client := awss3.NewFromConfig(cfg, func(o *awss3.Options) {
		o.UsePathStyle = true
	})

	result, err := client.ListObjectsV2(ctx, &awss3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(result.Contents))
	for _, obj := range result.Contents {
		keys = append(keys, *obj.Key)
	}

	return keys, nil
}

func getS3Object(ctx context.Context, endpoint, key string) ([]byte, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{
						URL:           endpoint,
						SigningRegion: region,
						Source:        aws.EndpointSourceCustom,
					}, nil
				},
			),
		),
	)
	if err != nil {
		return nil, err
	}

	client := awss3.NewFromConfig(cfg, func(o *awss3.Options) {
		o.UsePathStyle = true
	})

	result, err := client.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(testBucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()

	return io.ReadAll(result.Body)
}

func runSyncerCommand(ctx context.Context, command string, configFile string) (string, string, error) {
	cmd := exec.CommandContext(ctx, "go", "run", "../../main.go", command, "--config", configFile)
	stdout, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(stdout), string(exitErr.Stderr), exitErr
		}
		return string(stdout), "", err
	}
	return string(stdout), "", nil
}

func createSyncerConfig(romsDir string) syncer.Config {
	return syncer.Config{
		Username:   testUsername,
		RomsFolder: romsDir,
		Storage: syncer.Storage{
			S3: storage.S3Config{
				Enabled:                true,
				Bucket:                 testBucketName,
				CreateMissingResources: true,
			},
		},
		Sync: syncer.Sync{
			Roms:   true,
			Saves:  true,
			States: true,
		},
	}
}

func verifyFileUploaded(ctx context.Context, env *testEnvironment, expectedKey string, expectedContent []byte) error {
	objects, err := listS3Objects(ctx, env.endpoint)
	if err != nil {
		return err
	}

	found := false
	for _, key := range objects {
		if key == expectedKey {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("expected key %s not found in S3. Found keys: %v", expectedKey, objects)
	}

	content, err := getS3Object(ctx, env.endpoint, expectedKey)
	if err != nil {
		return err
	}

	if string(content) != string(expectedContent) {
		return fmt.Errorf("content mismatch for key %s", expectedKey)
	}

	return nil
}

func getFileContent(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

func createTestFile(dir, name, content string) (*fs.File, error) {
	filePath := filepath.Join(dir, name)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return nil, err
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	return fs.NewFile(filePath, info.ModTime()), nil
}
