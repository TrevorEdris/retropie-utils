package integration

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/TrevorEdris/retropie-utils/pkg/fs"
	"github.com/TrevorEdris/retropie-utils/pkg/log"
	"github.com/TrevorEdris/retropie-utils/tools/syncer/pkg/syncer"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sync Command Integration", func() {
	var (
		env *testEnvironment
		ctx context.Context
	)

	BeforeEach(func() {
		var err error
		env, err = setupTestEnvironment()
		Expect(err).NotTo(HaveOccurred())

		ctx = log.ToCtx(context.Background(), log.FromCtx(context.Background()))

		// Create S3 bucket
		err = createS3Bucket(ctx, env.endpoint)
		Expect(err).NotTo(HaveOccurred())

		// Wait a bit for bucket to be ready
		time.Sleep(1 * time.Second)
	})

	AfterEach(func() {
		err := teardownTestEnvironment(env)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Syncing files", func() {
		Context("when syncing ROMs", func() {
			It("uploads ROM files to S3 with correct path structure", func() {
				cfg := createSyncerConfig(env.tempRomsDir)
				cfg.Sync = syncer.Sync{
					Roms:   true,
					Saves:  false,
					States: false,
				}

				s, err := syncer.NewSyncer(ctx, cfg)
				Expect(err).NotTo(HaveOccurred())

				err = s.Sync(ctx)
				Expect(err).NotTo(HaveOccurred())

				// Verify files were uploaded
				// Expected path format: {remoteDir}/{username}/{file.Dir}/{file.Name}
				// remoteDir format: 2006/01/02/15 (year/month/day/hour)
				now := time.Now()
				remoteDir := now.Format("2006/01/02/15")

				expectedKeys := []string{
					fmt.Sprintf("%s/%s/gb/test-rom.gb", remoteDir, testUsername),
					fmt.Sprintf("%s/%s/gba/test-gba-rom.gba", remoteDir, testUsername),
					fmt.Sprintf("%s/%s/gbc/test-gbc-rom.gbc", remoteDir, testUsername),
				}

				objects, err := listS3Objects(ctx, env.endpoint)
				Expect(err).NotTo(HaveOccurred())
				Expect(objects).To(HaveLen(3))

				for _, expectedKey := range expectedKeys {
					Expect(objects).To(ContainElement(expectedKey))
				}

				// Verify content
				gbContent, err := getFileContent(filepath.Join(env.tempRomsDir, "gb", "test-rom.gb"))
				Expect(err).NotTo(HaveOccurred())

				s3Content, err := getS3Object(ctx, env.endpoint, expectedKeys[0])
				Expect(err).NotTo(HaveOccurred())
				Expect(s3Content).To(Equal(gbContent))
			})
		})

		Context("when syncing saves", func() {
			It("uploads save files to S3 with correct path structure", func() {
				cfg := createSyncerConfig(env.tempRomsDir)
				cfg.Sync = syncer.Sync{
					Roms:   false,
					Saves:  true,
					States: false,
				}

				s, err := syncer.NewSyncer(ctx, cfg)
				Expect(err).NotTo(HaveOccurred())

				err = s.Sync(ctx)
				Expect(err).NotTo(HaveOccurred())

				now := time.Now()
				remoteDir := now.Format("2006/01/02/15")

				expectedKeys := []string{
					fmt.Sprintf("%s/%s/gb/test-save.srm", remoteDir, testUsername),
					fmt.Sprintf("%s/%s/gba/test-gba-save.sav", remoteDir, testUsername),
				}

				objects, err := listS3Objects(ctx, env.endpoint)
				Expect(err).NotTo(HaveOccurred())
				Expect(objects).To(HaveLen(2))

				for _, expectedKey := range expectedKeys {
					Expect(objects).To(ContainElement(expectedKey))
				}
			})
		})

		Context("when syncing states", func() {
			It("uploads state files to S3 with correct path structure", func() {
				cfg := createSyncerConfig(env.tempRomsDir)
				cfg.Sync = syncer.Sync{
					Roms:   false,
					Saves:  false,
					States: true,
				}

				s, err := syncer.NewSyncer(ctx, cfg)
				Expect(err).NotTo(HaveOccurred())

				err = s.Sync(ctx)
				Expect(err).NotTo(HaveOccurred())

				now := time.Now()
				remoteDir := now.Format("2006/01/02/15")

				expectedKey := fmt.Sprintf("%s/%s/gb/test-state.state", remoteDir, testUsername)

				objects, err := listS3Objects(ctx, env.endpoint)
				Expect(err).NotTo(HaveOccurred())
				Expect(objects).To(HaveLen(1))
				Expect(objects).To(ContainElement(expectedKey))
			})
		})

		Context("when syncing all file types", func() {
			It("uploads ROMs, saves, and states to S3", func() {
				cfg := createSyncerConfig(env.tempRomsDir)
				cfg.Sync = syncer.Sync{
					Roms:   true,
					Saves:  true,
					States: true,
				}

				s, err := syncer.NewSyncer(ctx, cfg)
				Expect(err).NotTo(HaveOccurred())

				err = s.Sync(ctx)
				Expect(err).NotTo(HaveOccurred())

				now := time.Now()
				remoteDir := now.Format("2006/01/02/15")

				expectedKeys := []string{
					fmt.Sprintf("%s/%s/gb/test-rom.gb", remoteDir, testUsername),
					fmt.Sprintf("%s/%s/gb/test-save.srm", remoteDir, testUsername),
					fmt.Sprintf("%s/%s/gb/test-state.state", remoteDir, testUsername),
					fmt.Sprintf("%s/%s/gba/test-gba-rom.gba", remoteDir, testUsername),
					fmt.Sprintf("%s/%s/gba/test-gba-save.sav", remoteDir, testUsername),
					fmt.Sprintf("%s/%s/gbc/test-gbc-rom.gbc", remoteDir, testUsername),
				}

				objects, err := listS3Objects(ctx, env.endpoint)
				Expect(err).NotTo(HaveOccurred())
				Expect(objects).To(HaveLen(6))

				for _, expectedKey := range expectedKeys {
					Expect(objects).To(ContainElement(expectedKey))
				}
			})
		})

		Context("when no files match the file type", func() {
			It("completes successfully without errors", func() {
				// Create a directory with only non-matching files
				tempDir, err := os.MkdirTemp("", "syncer-no-files-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(tempDir)

				// Create a file that doesn't match any known type
				err = os.WriteFile(filepath.Join(tempDir, "readme.txt"), []byte("readme"), 0644)
				Expect(err).NotTo(HaveOccurred())

				cfg := createSyncerConfig(tempDir)
				cfg.Sync = syncer.Sync{
					Roms:   true,
					Saves:  true,
					States: true,
				}

				s, err := syncer.NewSyncer(ctx, cfg)
				Expect(err).NotTo(HaveOccurred())

				err = s.Sync(ctx)
				Expect(err).NotTo(HaveOccurred())

				// Verify no objects were uploaded
				objects, err := listS3Objects(ctx, env.endpoint)
				Expect(err).NotTo(HaveOccurred())
				Expect(objects).To(BeEmpty())
			})
		})

		Context("when syncing from subdirectories", func() {
			It("preserves directory structure in S3", func() {
				// Create nested directory structure
				nestedDir := filepath.Join(env.tempRomsDir, "nested", "level1", "level2")
				err := os.MkdirAll(nestedDir, 0755)
				Expect(err).NotTo(HaveOccurred())

				_, err = createTestFile(nestedDir, "nested-rom.gb", "nested content")
				Expect(err).NotTo(HaveOccurred())

				cfg := createSyncerConfig(env.tempRomsDir)
				cfg.Sync = syncer.Sync{
					Roms:   true,
					Saves:  false,
					States: false,
				}

				s, err := syncer.NewSyncer(ctx, cfg)
				Expect(err).NotTo(HaveOccurred())

				err = s.Sync(ctx)
				Expect(err).NotTo(HaveOccurred())

				now := time.Now()
				remoteDir := now.Format("2006/01/02/15")
				expectedKey := fmt.Sprintf("%s/%s/level2/nested-rom.gb", remoteDir, testUsername)

				objects, err := listS3Objects(ctx, env.endpoint)
				Expect(err).NotTo(HaveOccurred())
				Expect(objects).To(ContainElement(expectedKey))
			})
		})

		Context("when comparing file timestamps", func() {
			It("uploads file when S3 file does not exist", func() {
				// Create a new file that doesn't exist in S3
				_, err := createTestFile(filepath.Join(env.tempRomsDir, "gb"), "new-file.srm", "new save content")
				Expect(err).NotTo(HaveOccurred())

				cfg := createSyncerConfig(env.tempRomsDir)
				cfg.Sync = syncer.Sync{
					Roms:   false,
					Saves:  true,
					States: false,
				}

				s, err := syncer.NewSyncer(ctx, cfg)
				Expect(err).NotTo(HaveOccurred())

				err = s.Sync(ctx)
				Expect(err).NotTo(HaveOccurred())

				// Verify file was uploaded
				now := time.Now()
				remoteDir := now.Format("2006/01/02/15")
				expectedKey := fmt.Sprintf("%s/%s/gb/new-file.srm", remoteDir, testUsername)

				objects, err := listS3Objects(ctx, env.endpoint)
				Expect(err).NotTo(HaveOccurred())
				Expect(objects).To(ContainElement(expectedKey))

				// Verify content
				content, err := getS3Object(ctx, env.endpoint, expectedKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).To(Equal("new save content"))
			})

			It("uploads file when local file is newer than S3", func() {
				// First, upload a file to S3
				testFile, err := createTestFile(filepath.Join(env.tempRomsDir, "gb"), "timestamp-test.srm", "old content")
				Expect(err).NotTo(HaveOccurred())

				cfg := createSyncerConfig(env.tempRomsDir)
				cfg.Sync = syncer.Sync{
					Roms:   false,
					Saves:  true,
					States: false,
				}

				s, err := syncer.NewSyncer(ctx, cfg)
				Expect(err).NotTo(HaveOccurred())

				// First sync - uploads the file
				err = s.Sync(ctx)
				Expect(err).NotTo(HaveOccurred())

				// Wait a bit to ensure timestamp difference
				time.Sleep(100 * time.Millisecond)

				// Update the local file with new content and newer timestamp
				newContent := "new content"
				err = os.WriteFile(testFile.Absolute, []byte(newContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				// Update the file's modification time
				newTime := time.Now()
				err = os.Chtimes(testFile.Absolute, newTime, newTime)
				Expect(err).NotTo(HaveOccurred())

				// Re-read the file to get updated LastModified
				info, err := os.Stat(testFile.Absolute)
				Expect(err).NotTo(HaveOccurred())
				testFile = fs.NewFile(testFile.Absolute, info.ModTime())

				// Second sync - should upload because local is newer
				err = s.Sync(ctx)
				Expect(err).NotTo(HaveOccurred())

				// Verify the file in S3 has the new content
				now := time.Now()
				remoteDir := now.Format("2006/01/02/15")
				expectedKey := fmt.Sprintf("%s/%s/gb/timestamp-test.srm", remoteDir, testUsername)

				content, err := getS3Object(ctx, env.endpoint, expectedKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).To(Equal(newContent))
			})

			It("downloads file when S3 file is newer than local", func() {
				// Create a file locally
				testFile, err := createTestFile(filepath.Join(env.tempRomsDir, "gb"), "download-test.srm", "local old content")
				Expect(err).NotTo(HaveOccurred())

				cfg := createSyncerConfig(env.tempRomsDir)
				cfg.Sync = syncer.Sync{
					Roms:   false,
					Saves:  true,
					States: false,
				}

				s, err := syncer.NewSyncer(ctx, cfg)
				Expect(err).NotTo(HaveOccurred())

				// First sync - uploads the file
				err = s.Sync(ctx)
				Expect(err).NotTo(HaveOccurred())

				// Wait a bit
				time.Sleep(100 * time.Millisecond)

				// Manually upload a newer version to S3 (simulating another device)
				now := time.Now()
				remoteDir := now.Format("2006/01/02/15")
				expectedKey := fmt.Sprintf("%s/%s/gb/download-test.srm", remoteDir, testUsername)

				// Upload newer content directly to S3
				s3Content := "s3 newer content"
				cfg2, err := config.LoadDefaultConfig(ctx,
					config.WithEndpointResolverWithOptions(
						aws.EndpointResolverWithOptionsFunc(
							func(service, region string, options ...interface{}) (aws.Endpoint, error) {
								return aws.Endpoint{
									URL:           env.endpoint,
									SigningRegion: region,
									Source:        aws.EndpointSourceCustom,
								}, nil
							},
						),
					),
				)
				Expect(err).NotTo(HaveOccurred())

				s3Client := awss3.NewFromConfig(cfg2, func(o *awss3.Options) {
					o.UsePathStyle = true
				})

				_, err = s3Client.PutObject(ctx, &awss3.PutObjectInput{
					Bucket: aws.String(testBucketName),
					Key:    aws.String(expectedKey),
					Body:   bytes.NewReader([]byte(s3Content)),
				})
				Expect(err).NotTo(HaveOccurred())

				// Wait a bit more to ensure S3 timestamp is newer
				time.Sleep(100 * time.Millisecond)

				// Make local file older by setting its modification time to the past
				pastTime := time.Now().Add(-1 * time.Hour)
				err = os.Chtimes(testFile.Absolute, pastTime, pastTime)
				Expect(err).NotTo(HaveOccurred())

				// Re-read the file to get updated LastModified
				info, err := os.Stat(testFile.Absolute)
				Expect(err).NotTo(HaveOccurred())
				testFile = fs.NewFile(testFile.Absolute, info.ModTime())

				// Second sync - should download because S3 is newer
				err = s.Sync(ctx)
				Expect(err).NotTo(HaveOccurred())

				// Verify local file was updated with S3 content
				localContent, err := os.ReadFile(testFile.Absolute)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(localContent)).To(Equal(s3Content))
			})
		})
	})

	Describe("Bucket creation", func() {
		Context("when bucket does not exist and CreateMissingResources is enabled", func() {
			It("creates the bucket automatically", func() {
				// Use a new bucket name and empty directory to avoid syncing files
				newBucketName := "new-test-bucket"
				emptyDir, err := os.MkdirTemp("", "syncer-empty-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(emptyDir)

				cfg := createSyncerConfig(emptyDir)
				cfg.Storage.S3.Bucket = newBucketName
				cfg.Storage.S3.CreateMissingResources = true

				s, err := syncer.NewSyncer(ctx, cfg)
				Expect(err).NotTo(HaveOccurred())

				err = s.Sync(ctx)
				Expect(err).NotTo(HaveOccurred())

				// Verify bucket was created by attempting to list objects
				// (this would fail if bucket doesn't exist)
				objects, err := listS3ObjectsWithBucket(ctx, env.endpoint, newBucketName)
				Expect(err).NotTo(HaveOccurred())
				Expect(objects).To(BeEmpty()) // Bucket exists but is empty since directory is empty
			})
		})
	})
})
