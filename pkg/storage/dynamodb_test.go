package storage_test

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/TrevorEdris/retropie-utils/pkg/fs"
	"github.com/TrevorEdris/retropie-utils/pkg/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var _ = Describe("DynamoDB", func() {
	Context("DynamoDB client", func() {
		createLocalstackContainer := func() testcontainers.Container {
			ctx := context.Background()

			// Create LocalStack container request with DynamoDB service
			req := testcontainers.ContainerRequest{
				Image:        "localstack/localstack:latest",
				ExposedPorts: []string{"4566/tcp"},
				Env: map[string]string{
					"SERVICES": "dynamodb",
				},
				WaitingFor: wait.ForHTTP("/_localstack/health").WithPort("4566/tcp").WithStatusCodeMatcher(func(status int) bool {
					return status == http.StatusOK
				}),
			}

			// Start the LocalStack container
			localstackC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
				ContainerRequest: req,
				Started:          true,
			})
			if err != nil {
				log.Fatalf("Failed to start LocalStack container: %v", err)
			}

			return localstackC
		}

		setAwsEnv := func(endpoint string) {
			err := os.Setenv("AWS_ENDPOINT", endpoint)
			if err != nil {
				panic(err)
			}
			err = os.Setenv("AWS_ACCESS_KEY_ID", "fakekey")
			if err != nil {
				panic(err)
			}
			err = os.Setenv("AWS_SECRET_ACCESS_KEY", "fakesecret")
			if err != nil {
				panic(err)
			}
		}

		unsetAwsEnv := func() {
			err := os.Unsetenv("AWS_ENDPOINT")
			if err != nil {
				panic(err)
			}
			err = os.Unsetenv("AWS_ACCESS_KEY_ID")
			if err != nil {
				panic(err)
			}
			err = os.Unsetenv("AWS_SECRET_ACCESS_KEY")
			if err != nil {
				panic(err)
			}
		}

		getContainerEndpoint := func(container testcontainers.Container) string {
			p, err := container.MappedPort(context.TODO(), "4566")
			if err != nil {
				panic(err)
			}
			h, err := container.Host(context.TODO())
			if err != nil {
				panic(err)
			}
			return "http://" + h + ":" + p.Port()
		}

		It("creates necessary base resources", func() {
			container := createLocalstackContainer()
			defer func() {
				err := container.Terminate(context.TODO())
				if err != nil {
					panic(err)
				}
			}()
			endpoint := getContainerEndpoint(container)
			setAwsEnv(endpoint)
			defer unsetAwsEnv()

			client, err := storage.NewDynamoDBClient(context.TODO(), storage.DynamoDBConfig{
				Enabled:                true,
				TableName:              "test-table",
				CreateMissingResources: true,
			}, "test-username")
			Expect(err).NotTo(HaveOccurred())

			err = client.Init(context.TODO())
			Expect(err).NotTo(HaveOccurred())

			// The 2nd call will check if the table exists again
			err = client.Init(context.TODO())
			Expect(err).NotTo(HaveOccurred())
		})

		It("stores and retrieves file metadata", func() {
			container := createLocalstackContainer()
			defer func() {
				err := container.Terminate(context.TODO())
				if err != nil {
					panic(err)
				}
			}()
			endpoint := getContainerEndpoint(container)
			setAwsEnv(endpoint)
			defer unsetAwsEnv()

			client, err := storage.NewDynamoDBClient(context.TODO(), storage.DynamoDBConfig{
				Enabled:                true,
				TableName:              "test-table",
				CreateMissingResources: true,
			}, "test-username")
			Expect(err).NotTo(HaveOccurred())

			err = client.Init(context.TODO())
			Expect(err).NotTo(HaveOccurred())

			// Create a test file
			tmpFile, err := os.CreateTemp("", "testfile.sav")
			Expect(err).NotTo(HaveOccurred())
			defer os.Remove(tmpFile.Name())
			err = tmpFile.Close()
			Expect(err).NotTo(HaveOccurred())

			testFile := &fs.File{
				Dir:          "gba",
				Absolute:     tmpFile.Name(),
				Name:         "Pokemon Fire Red.sav",
				LastModified: time.Now(),
				FileType:     fs.Save,
			}

			s3Location := "2024/01/17/12/test-username/gba/Pokemon Fire Red.sav"
			err = client.StoreFileMetadata(context.TODO(), s3Location, testFile, "2024/01/17/12")
			Expect(err).NotTo(HaveOccurred())

			// Retrieve the metadata
			metadata, err := client.GetFileMetadataByFile(context.TODO(), testFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(metadata).NotTo(BeNil())
			Expect(metadata.S3Location).To(Equal(s3Location))
			Expect(metadata.OriginalFileName).To(Equal("Pokemon Fire Red.sav"))
			Expect(metadata.FileDir).To(Equal("gba"))
			Expect(metadata.Username).To(Equal("test-username"))
			Expect(metadata.FileType).To(Equal("save"))
			Expect(metadata.FileIdentifier).To(Equal("test-username#gba#pokemon_fire_red.sav"))
		})

		It("updates existing metadata when file is re-uploaded", func() {
			container := createLocalstackContainer()
			defer func() {
				err := container.Terminate(context.TODO())
				if err != nil {
					panic(err)
				}
			}()
			endpoint := getContainerEndpoint(container)
			setAwsEnv(endpoint)
			defer unsetAwsEnv()

			client, err := storage.NewDynamoDBClient(context.TODO(), storage.DynamoDBConfig{
				Enabled:                true,
				TableName:              "test-table",
				CreateMissingResources: true,
			}, "test-username")
			Expect(err).NotTo(HaveOccurred())

			err = client.Init(context.TODO())
			Expect(err).NotTo(HaveOccurred())

			// Create a test file
			tmpFile, err := os.CreateTemp("", "testfile.sav")
			Expect(err).NotTo(HaveOccurred())
			defer os.Remove(tmpFile.Name())
			err = tmpFile.Close()
			Expect(err).NotTo(HaveOccurred())

			testFile := &fs.File{
				Dir:          "gba",
				Absolute:     tmpFile.Name(),
				Name:         "Pokemon Fire Red.sav",
				LastModified: time.Now(),
				FileType:     fs.Save,
			}

			// Store first version
			s3Location1 := "2024/01/17/12/test-username/gba/Pokemon Fire Red.sav"
			err = client.StoreFileMetadata(context.TODO(), s3Location1, testFile, "2024/01/17/12")
			Expect(err).NotTo(HaveOccurred())

			// Wait a bit to ensure different timestamp
			time.Sleep(10 * time.Millisecond)

			// Store second version
			s3Location2 := "2024/01/17/13/test-username/gba/Pokemon Fire Red.sav"
			err = client.StoreFileMetadata(context.TODO(), s3Location2, testFile, "2024/01/17/13")
			Expect(err).NotTo(HaveOccurred())

			// Retrieve the metadata - should have latest location
			metadata, err := client.GetFileMetadataByFile(context.TODO(), testFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(metadata).NotTo(BeNil())
			Expect(metadata.S3Location).To(Equal(s3Location2))
			Expect(metadata.LastModifiedTime).To(BeNumerically(">", metadata.CreatedAt))
		})

		It("returns nil when file metadata does not exist", func() {
			container := createLocalstackContainer()
			defer func() {
				err := container.Terminate(context.TODO())
				if err != nil {
					panic(err)
				}
			}()
			endpoint := getContainerEndpoint(container)
			setAwsEnv(endpoint)
			defer unsetAwsEnv()

			client, err := storage.NewDynamoDBClient(context.TODO(), storage.DynamoDBConfig{
				Enabled:                true,
				TableName:              "test-table",
				CreateMissingResources: true,
			}, "test-username")
			Expect(err).NotTo(HaveOccurred())

			err = client.Init(context.TODO())
			Expect(err).NotTo(HaveOccurred())

			// Try to retrieve non-existent file
			testFile := &fs.File{
				Dir:      "gba",
				Name:     "NonExistent.sav",
				FileType: fs.Save,
			}

			metadata, err := client.GetFileMetadataByFile(context.TODO(), testFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(metadata).To(BeNil())
		})
	})
})

