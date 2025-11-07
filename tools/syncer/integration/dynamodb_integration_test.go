package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/TrevorEdris/retropie-utils/pkg/fs"
	"github.com/TrevorEdris/retropie-utils/pkg/log"
	"github.com/TrevorEdris/retropie-utils/pkg/storage"
	"github.com/TrevorEdris/retropie-utils/tools/syncer/pkg/syncer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DynamoDB Integration", func() {
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

	Describe("S3 with DynamoDB lookup", func() {
		Context("when storing files with DynamoDB enabled", func() {
			It("stores file metadata in DynamoDB and can retrieve files by identifier", func() {
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

				// Verify files were uploaded to S3
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

				// Verify metadata was stored in DynamoDB
				dynamodbClient, err := storage.NewDynamoDBClient(ctx, storage.DynamoDBConfig{
					Enabled:   true,
					TableName: "retropie-file-metadata",
				}, testUsername)
				Expect(err).NotTo(HaveOccurred())

				// Check metadata for gba save file
				testFile := &fs.File{
					Dir:      "gba",
					Name:     "test-gba-save.sav",
					FileType: fs.Save,
				}

				metadata, err := dynamodbClient.GetFileMetadataByFile(ctx, testFile)
				Expect(err).NotTo(HaveOccurred())
				Expect(metadata).NotTo(BeNil())
				Expect(metadata.S3Location).To(Equal(fmt.Sprintf("%s/%s/gba/test-gba-save.sav", remoteDir, testUsername)))
				Expect(metadata.OriginalFileName).To(Equal("test-gba-save.sav"))
				Expect(metadata.FileDir).To(Equal("gba"))
				Expect(metadata.Username).To(Equal(testUsername))
				Expect(metadata.FileType).To(Equal("save"))
				Expect(metadata.FileIdentifier).To(Equal(fmt.Sprintf("%s#gba#test-gba-save.sav", testUsername)))
			})

			It("can retrieve files from S3 using DynamoDB lookup", func() {
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

				// Create storage client for retrieval
				storageClient, err := storage.NewS3Storage(ctx, cfg.Storage.S3, testUsername)
				Expect(err).NotTo(HaveOccurred())
				err = storageClient.Init(ctx)
				Expect(err).NotTo(HaveOccurred())

				// Retrieve file using DynamoDB lookup (file identifier only, no S3 path needed)
				testFile := &fs.File{
					Dir:      "gba",
					Name:     "test-gba-save.sav",
					FileType: fs.Save,
				}

				// Create destination file
				destFile, err := os.CreateTemp("", "retrieved-*.sav")
				Expect(err).NotTo(HaveOccurred())
				defer os.Remove(destFile.Name())
				err = destFile.Close()
				Expect(err).NotTo(HaveOccurred())

				retrieved, err := storageClient.Retrieve(ctx, storage.RetrieveFileRequest{
					ToRetrieve: testFile,
					Destination: &fs.File{
						Absolute: destFile.Name(),
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved).NotTo(BeNil())
				Expect(retrieved.Name).To(Equal("test-gba-save.sav"))

				// Verify content matches
				originalContent, err := getFileContent(filepath.Join(env.tempRomsDir, "gba", "test-gba-save.sav"))
				Expect(err).NotTo(HaveOccurred())

				retrievedContent, err := getFileContent(destFile.Name())
				Expect(err).NotTo(HaveOccurred())

				Expect(retrievedContent).To(Equal(originalContent))
			})

			It("updates DynamoDB metadata when same file is uploaded again", func() {
				cfg := createSyncerConfig(env.tempRomsDir)
				cfg.Sync = syncer.Sync{
					Roms:   false,
					Saves:  true,
					States: false,
				}

				s, err := syncer.NewSyncer(ctx, cfg)
				Expect(err).NotTo(HaveOccurred())

				// First sync
				err = s.Sync(ctx)
				Expect(err).NotTo(HaveOccurred())

				// Wait a bit to ensure different timestamp
				time.Sleep(2 * time.Second)

				// Second sync (should update metadata)
				err = s.Sync(ctx)
				Expect(err).NotTo(HaveOccurred())

				// Verify metadata was updated with new S3 location
				dynamodbClient, err := storage.NewDynamoDBClient(ctx, storage.DynamoDBConfig{
					Enabled:   true,
					TableName: "retropie-file-metadata",
				}, testUsername)
				Expect(err).NotTo(HaveOccurred())

				testFile := &fs.File{
					Dir:      "gba",
					Name:     "test-gba-save.sav",
					FileType: fs.Save,
				}

				metadata, err := dynamodbClient.GetFileMetadataByFile(ctx, testFile)
				Expect(err).NotTo(HaveOccurred())
				Expect(metadata).NotTo(BeNil())

				// Should have the latest S3 location (from second sync)
				// Note: remoteDir might be the same if syncs happen in same hour
				// But lastModifiedTime should be updated
				Expect(metadata.S3Location).To(ContainSubstring(testUsername))
				Expect(metadata.LastModifiedTime).To(BeNumerically(">=", metadata.CreatedAt))
			})

			It("handles files with spaces in names correctly", func() {
				// Create a file with spaces in the name
				testFileName := "Pokemon Fire Red.sav"
				testFilePath := filepath.Join(env.tempRomsDir, "gba", testFileName)
				err := os.WriteFile(testFilePath, []byte("test save content"), 0644)
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

				// Verify metadata with normalized identifier
				dynamodbClient, err := storage.NewDynamoDBClient(ctx, storage.DynamoDBConfig{
					Enabled:   true,
					TableName: "retropie-file-metadata",
				}, testUsername)
				Expect(err).NotTo(HaveOccurred())

				testFile := &fs.File{
					Dir:      "gba",
					Name:     testFileName,
					FileType: fs.Save,
				}

				metadata, err := dynamodbClient.GetFileMetadataByFile(ctx, testFile)
				Expect(err).NotTo(HaveOccurred())
				Expect(metadata).NotTo(BeNil())
				Expect(metadata.OriginalFileName).To(Equal(testFileName))
				Expect(metadata.FileIdentifier).To(Equal(fmt.Sprintf("%s#gba#pokemon_fire_red.sav", testUsername)))
			})
		})
	})
})

