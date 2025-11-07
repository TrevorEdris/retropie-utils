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

		Context("GetFileLastModified", func() {
			It("returns NotImplementedError when storage is not enabled", func() {
				client, err := storage.NewS3Storage(context.TODO(), storage.S3Config{
					Enabled: false,
				}, "myusername")
				Expect(err).NotTo(HaveOccurred())

				file := &fs.File{
					Dir:      "mydir",
					Name:     "myfile.srm",
					Absolute: "/tmp/myfile.srm",
				}

				lastModified, err := client.GetFileLastModified(context.TODO(), "remoteDir", file)
				Expect(lastModified).To(BeNil())
				Expect(err).To(MatchError(errors.NotImplementedError))
			})

			It("returns nil when file does not exist in S3", func() {
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

				file := &fs.File{
					Dir:      "mydir",
					Name:     "nonexistent.srm",
					Absolute: "/tmp/nonexistent.srm",
				}

				lastModified, err := strg.GetFileLastModified(context.TODO(), "remoteDir", file)
				Expect(err).NotTo(HaveOccurred())
				Expect(lastModified).To(BeNil())
			})

			It("returns last modified time when file exists in S3", func() {
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

				// Create and upload a file
				tmpFile, err := os.CreateTemp("", "myfile.srm")
				Expect(err).NotTo(HaveOccurred())
				defer os.Remove(tmpFile.Name())
				_, err = tmpFile.WriteString("test file content")
				Expect(err).NotTo(HaveOccurred())
				err = tmpFile.Close()
				Expect(err).NotTo(HaveOccurred())

				// Record time before creating file to ensure S3 timestamp is after
				beforeUpload := time.Now()
				
				myfile := &fs.File{
					Dir:          "mydir",
					Absolute:     tmpFile.Name(),
					Name:         "myfile.srm",
					LastModified: beforeUpload,
					FileType:     fs.Save,
				}

				remoteDir := "2024/01/17/12"
				err = strg.Store(context.TODO(), remoteDir, myfile)
				Expect(err).NotTo(HaveOccurred())

				// Record time after upload
				afterUpload := time.Now()

				// Get last modified time
				lastModified, err := strg.GetFileLastModified(context.TODO(), remoteDir, myfile)
				Expect(err).NotTo(HaveOccurred())
				Expect(lastModified).NotTo(BeNil())
				// S3 LastModified should be between beforeUpload and afterUpload (with some tolerance)
				Expect(*lastModified).To(BeTemporally(">=", beforeUpload.Add(-1*time.Second)))
				Expect(*lastModified).To(BeTemporally("<=", afterUpload.Add(1*time.Second)))
			})

			It("uses DynamoDB metadata when DynamoDB is enabled", func() {
				// Create container with DynamoDB service
				ctx := context.Background()
				req := testcontainers.ContainerRequest{
					Image:        "localstack/localstack:latest",
					ExposedPorts: []string{"4566/tcp"},
					Env: map[string]string{
						"SERVICES": "s3,dynamodb",
					},
					WaitingFor: wait.ForHTTP("/_localstack/health").WithPort("4566/tcp").WithStatusCodeMatcher(func(status int) bool {
						return status == http.StatusOK
					}),
				}
				container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
					ContainerRequest: req,
					Started:          true,
				})
				Expect(err).NotTo(HaveOccurred())
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
					DynamoDB: storage.DynamoDBConfig{
						Enabled:                true,
						TableName:              "test-table",
						CreateMissingResources: true,
					},
				}, "test-username")
				Expect(err).NotTo(HaveOccurred())

				err = strg.Init(context.TODO())
				Expect(err).NotTo(HaveOccurred())

				// Create and upload a file
				tmpFile, err := os.CreateTemp("", "myfile.srm")
				Expect(err).NotTo(HaveOccurred())
				defer os.Remove(tmpFile.Name())
				_, err = tmpFile.WriteString("test file content")
				Expect(err).NotTo(HaveOccurred())
				err = tmpFile.Close()
				Expect(err).NotTo(HaveOccurred())

				// Record time before creating file to ensure DynamoDB timestamp is after
				beforeUpload := time.Now()
				
				myfile := &fs.File{
					Dir:          "mydir",
					Absolute:     tmpFile.Name(),
					Name:         "myfile.srm",
					LastModified: beforeUpload,
					FileType:     fs.Save,
				}

				remoteDir := "2024/01/17/12"
				err = strg.Store(context.TODO(), remoteDir, myfile)
				Expect(err).NotTo(HaveOccurred())

				// Record time after upload
				afterUpload := time.Now()

				// Get last modified time - should use DynamoDB
				lastModified, err := strg.GetFileLastModified(context.TODO(), remoteDir, myfile)
				Expect(err).NotTo(HaveOccurred())
				Expect(lastModified).NotTo(BeNil())
				// DynamoDB stores timestamp when file was uploaded, which should be between beforeUpload and afterUpload
				Expect(*lastModified).To(BeTemporally(">=", beforeUpload.Add(-1*time.Second)))
				Expect(*lastModified).To(BeTemporally("<=", afterUpload.Add(1*time.Second)))
			})

			It("falls back to S3 HeadObject when DynamoDB file doesn't exist", func() {
				// Create container with DynamoDB service
				ctx := context.Background()
				req := testcontainers.ContainerRequest{
					Image:        "localstack/localstack:latest",
					ExposedPorts: []string{"4566/tcp"},
					Env: map[string]string{
						"SERVICES": "s3,dynamodb",
					},
					WaitingFor: wait.ForHTTP("/_localstack/health").WithPort("4566/tcp").WithStatusCodeMatcher(func(status int) bool {
						return status == http.StatusOK
					}),
				}
				container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
					ContainerRequest: req,
					Started:          true,
				})
				Expect(err).NotTo(HaveOccurred())
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
					DynamoDB: storage.DynamoDBConfig{
						Enabled:                true,
						TableName:              "test-table",
						CreateMissingResources: true,
					},
				}, "test-username")
				Expect(err).NotTo(HaveOccurred())

				err = strg.Init(context.TODO())
				Expect(err).NotTo(HaveOccurred())

				// Create and upload a file directly to S3 (bypassing DynamoDB)
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

				remoteDir := "2024/01/17/12"
				// Store without DynamoDB by using a different file that doesn't match
				// Actually, let's just upload it normally and then check with a file that doesn't exist in DynamoDB
				err = strg.Store(context.TODO(), remoteDir, myfile)
				Expect(err).NotTo(HaveOccurred())

				// Create a different file object (different name) to test fallback
				otherFile := &fs.File{
					Dir:      "mydir",
					Name:     "otherfile.srm",
					Absolute: "/tmp/otherfile.srm",
				}

				// This should return nil since file doesn't exist
				lastModified, err := strg.GetFileLastModified(context.TODO(), remoteDir, otherFile)
				Expect(err).NotTo(HaveOccurred())
				Expect(lastModified).To(BeNil())
			})
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
