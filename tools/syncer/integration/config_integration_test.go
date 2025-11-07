package integration

import (
	"os"
	"path/filepath"

	"github.com/TrevorEdris/retropie-utils/pkg/storage"
	"github.com/TrevorEdris/retropie-utils/tools/syncer/pkg/syncer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config Command Integration", func() {

	Describe("Config validation", func() {
		Context("with valid config", func() {
			It("passes validation", func() {
				tempDir, err := os.MkdirTemp("", "syncer-config-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(tempDir)

				cfg := syncer.Config{
					Username:   "validuser123",
					RomsFolder: tempDir,
					Storage: syncer.Storage{
						S3: storage.S3Config{
							Enabled: true,
							Bucket:  "test-bucket",
						},
					},
					Sync: syncer.Sync{
						Roms:   true,
						Saves:  true,
						States: true,
					},
				}

				err = syncer.ValidateConfig(&cfg)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("with invalid username", func() {
			It("fails validation for default username", func() {
				cfg := syncer.Config{
					Username: syncer.DefaultUsername,
					Storage: syncer.Storage{
						S3: storage.S3Config{
							Enabled: true,
							Bucket:  "test-bucket",
						},
					},
				}

				err := syncer.ValidateConfig(&cfg)
				Expect(err).To(HaveOccurred())
			})

			It("fails validation for too short username", func() {
				cfg := syncer.Config{
					Username: "ab",
					Storage: syncer.Storage{
						S3: storage.S3Config{
							Enabled: true,
							Bucket:  "test-bucket",
						},
					},
				}

				err := syncer.ValidateConfig(&cfg)
				Expect(err).To(HaveOccurred())
			})

			It("fails validation for invalid characters", func() {
				cfg := syncer.Config{
					Username: "user/name@invalid",
					Storage: syncer.Storage{
						S3: storage.S3Config{
							Enabled: true,
							Bucket:  "test-bucket",
						},
					},
				}

				err := syncer.ValidateConfig(&cfg)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with missing required fields", func() {
			It("fails validation when username is empty", func() {
				cfg := syncer.Config{
					Storage: syncer.Storage{
						S3: storage.S3Config{
							Enabled: true,
							Bucket:  "test-bucket",
						},
					},
				}

				err := syncer.ValidateConfig(&cfg)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Config file operations", func() {
		Context("when creating example config", func() {
			It("creates example config file in specified directory", func() {
				tempDir, err := os.MkdirTemp("", "syncer-example-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(tempDir)

				err = syncer.CreateExample(tempDir)
				Expect(err).NotTo(HaveOccurred())

				examplePath := filepath.Join(tempDir, "config.example.yaml")
				Expect(examplePath).To(BeAnExistingFile())

				// Verify file content
				content, err := os.ReadFile(examplePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(content).To(ContainSubstring("username:"))
				Expect(content).To(ContainSubstring("storage:"))
				Expect(content).To(ContainSubstring("sync:"))
			})
		})

		Context("when validating config file", func() {
			It("validates a valid config file", func() {
				tempDir, err := os.MkdirTemp("", "syncer-validate-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(tempDir)

				configFile := filepath.Join(tempDir, "config.yaml")
				configContent := `username: validuser123
romsFolder: /tmp/roms
storage:
  s3:
    enabled: true
    bucket: test-bucket
sync:
  roms: true
  saves: true
  states: true
`
				err = os.WriteFile(configFile, []byte(configContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				err = syncer.ValidateConfigFile(configFile)
				Expect(err).NotTo(HaveOccurred())
			})

			It("fails validation for invalid config file", func() {
				tempDir, err := os.MkdirTemp("", "syncer-validate-invalid-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(tempDir)

				configFile := filepath.Join(tempDir, "config.yaml")
				configContent := `username: DEFAULT_USERNAME_CHANGE_THIS_VALUE
romsFolder: /tmp/roms
storage:
  s3:
    enabled: true
    bucket: test-bucket
`
				err = os.WriteFile(configFile, []byte(configContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				err = syncer.ValidateConfigFile(configFile)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
