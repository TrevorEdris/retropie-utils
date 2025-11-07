package storage_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/TrevorEdris/retropie-utils/pkg/errors"
	"github.com/TrevorEdris/retropie-utils/pkg/fs"
	"github.com/TrevorEdris/retropie-utils/pkg/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var _ = Describe("S3", func() {
	Context("storage is not enabled", func() {
		It("everything is a no-op", func() {
			client, err := storage.NewS3Storage(context.TODO(), storage.S3Config{
				Enabled: false,
			}, "myusername")
			Expect(err).NotTo(HaveOccurred())
			err = client.Store(context.TODO(), "", nil)
			Expect(err).NotTo(HaveOccurred())
			err = client.StoreAll(context.TODO(), "", nil)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("storage is enabled", func() {

		createLocalstackContainer := func() testcontainers.Container {
			ctx := context.Background()

			// Create LocalStack container request
			req := testcontainers.ContainerRequest{
				Image:        "localstack/localstack:latest",
				ExposedPorts: []string{"4566/tcp"},
				Env: map[string]string{
					"SERVICES": "s3",
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
			return fmt.Sprintf("http://%s:%s", h, p.Port())
		}

		It("retrieve is not implemented", func() {
			client, err := storage.NewS3Storage(context.TODO(), storage.S3Config{
				Enabled: true,
			}, "myusername")
			Expect(err).NotTo(HaveOccurred())
			f, err := client.Retrieve(context.TODO(), storage.RetrieveFileRequest{})
			Expect(f).To(BeNil())
			Expect(err).To(MatchError(errors.NotImplementedError))
		})

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

			storage, err := storage.NewS3Storage(context.TODO(), storage.S3Config{
				Enabled:                true,
				Bucket:                 "test-bucket",
				CreateMissingResources: true,
			}, "test-username")
			Expect(err).NotTo(HaveOccurred())

			err = storage.Init(context.TODO())
			Expect(err).NotTo(HaveOccurred())

			// The 2nd call will check if the bucket exists again
			err = storage.Init(context.TODO())
			Expect(err).NotTo(HaveOccurred())
		})

		It("has username prepended to the path", func() {
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

			strg, err := storage.NewS3Storage(context.TODO(), storage.S3Config{
				Enabled:                true,
				Bucket:                 "test-bucket",
				CreateMissingResources: true,
			}, "test-username")
			Expect(err).NotTo(HaveOccurred())

			err = strg.Init(context.TODO())
			Expect(err).NotTo(HaveOccurred())

			// Create a temporary file for testing
			tmpFile, err := os.CreateTemp("", "myfile.srm")
			Expect(err).NotTo(HaveOccurred())
			defer os.Remove(tmpFile.Name())
			_, err = tmpFile.WriteString("test file content")
			Expect(err).NotTo(HaveOccurred())
			err = tmpFile.Close()
			Expect(err).NotTo(HaveOccurred())

			myfile := &fs.File{
				Dir:          "mydir",
				Absolute:     tmpFile.Name(),
				Name:         "myfile.srm",
				LastModified: time.Now(),
				FileType:     fs.Save,
			}
			err = strg.Store(context.TODO(), "roms/gbc", myfile)
			Expect(err).NotTo(HaveOccurred())

			// Create a temporary file for the destination
			destFile, err := os.CreateTemp("", "retrieved_myfile.srm")
			Expect(err).NotTo(HaveOccurred())
			defer os.Remove(destFile.Name())
			err = destFile.Close()
			Expect(err).NotTo(HaveOccurred())

			retrieved, err := strg.Retrieve(context.TODO(), storage.RetrieveFileRequest{
				ToRetrieve: &fs.File{
					Dir:  "roms/gbc/mydir",
					Name: "myfile.srm",
				},
				Destination: &fs.File{
					Absolute: destFile.Name(),
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.Name).To(Equal(myfile.Name))
		})
	})
})
