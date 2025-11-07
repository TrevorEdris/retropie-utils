package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/TrevorEdris/retropie-utils/pkg/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("CLI Integration", func() {
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

	Describe("sync command", func() {
		Context("when running sync command with valid config", func() {
			It("successfully syncs files", func() {
				// Build the syncer binary
				syncerPath, err := gexec.Build("github.com/TrevorEdris/retropie-utils/tools/syncer")
				Expect(err).NotTo(HaveOccurred())
				defer gexec.CleanupBuildArtifacts()

				// Run sync command
				cmd := exec.Command(syncerPath, "sync", "--config", env.configFile)
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				// Wait for command to complete (with timeout)
				Eventually(session, 30*time.Second).Should(gexec.Exit(0))

				// Verify files were uploaded
				objects, err := listS3Objects(ctx, env.endpoint)
				Expect(err).NotTo(HaveOccurred())
				Expect(objects).NotTo(BeEmpty())
			})
		})

		Context("when running sync command with invalid config", func() {
			It("exits with error", func() {
				// Create invalid config file
				invalidConfig, err := os.CreateTemp("", "invalid-config-*.yaml")
				Expect(err).NotTo(HaveOccurred())
				defer os.Remove(invalidConfig.Name())

				invalidContent := `username: DEFAULT_USERNAME_CHANGE_THIS_VALUE
romsFolder: /tmp/roms
storage:
  s3:
    enabled: true
    bucket: test-bucket
`
				err = os.WriteFile(invalidConfig.Name(), []byte(invalidContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				// Build the syncer binary
				syncerPath, err := gexec.Build("github.com/TrevorEdris/retropie-utils/tools/syncer")
				Expect(err).NotTo(HaveOccurred())
				defer gexec.CleanupBuildArtifacts()

				// Run sync command
				cmd := exec.Command(syncerPath, "sync", "--config", invalidConfig.Name())
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				// Command should exit with error
				Eventually(session, 10*time.Second).Should(gexec.Exit())
				Expect(session.ExitCode()).NotTo(Equal(0))
			})
		})
	})

	Describe("config command", func() {
		Context("when validating a valid config file", func() {
			It("succeeds", func() {
				// Build the syncer binary
				syncerPath, err := gexec.Build("github.com/TrevorEdris/retropie-utils/tools/syncer")
				Expect(err).NotTo(HaveOccurred())
				defer gexec.CleanupBuildArtifacts()

				// Run config validate command
				cmd := exec.Command(syncerPath, "config", "--config", env.configFile)
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				// Wait for command to complete
				Eventually(session, 10*time.Second).Should(gexec.Exit(0))
				Expect(session.Out.Contents()).To(ContainSubstring("Validation passed"))
			})
		})

		Context("when validating an invalid config file", func() {
			It("fails with appropriate error", func() {
				// Create invalid config
				invalidConfig, err := os.CreateTemp("", "invalid-config-*.yaml")
				Expect(err).NotTo(HaveOccurred())
				defer os.Remove(invalidConfig.Name())

				invalidContent := `username: DEFAULT_USERNAME_CHANGE_THIS_VALUE
romsFolder: /tmp/roms
storage:
  s3:
    enabled: true
    bucket: test-bucket
`
				err = os.WriteFile(invalidConfig.Name(), []byte(invalidContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				// Build the syncer binary
				syncerPath, err := gexec.Build("github.com/TrevorEdris/retropie-utils/tools/syncer")
				Expect(err).NotTo(HaveOccurred())
				defer gexec.CleanupBuildArtifacts()

				// Run config validate command
				cmd := exec.Command(syncerPath, "config", "--config", invalidConfig.Name())
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				// Command should exit with error
				Eventually(session, 10*time.Second).Should(gexec.Exit())
				Expect(session.ExitCode()).NotTo(Equal(0))
				// Check both stdout and stderr for validation error message
				output := string(session.Out.Contents()) + string(session.Err.Contents())
				Expect(output).To(ContainSubstring("Validation"))
			})
		})
	})

	Describe("init command", func() {
		Context("when initializing example config", func() {
			It("creates example config file", func() {
				tempDir, err := os.MkdirTemp("", "syncer-init-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(tempDir)

				// Build the syncer binary
				syncerPath, err := gexec.Build("github.com/TrevorEdris/retropie-utils/tools/syncer")
				Expect(err).NotTo(HaveOccurred())
				defer gexec.CleanupBuildArtifacts()

				// Change to temp directory to test init command
				// Note: The init command uses $HOME/.syncer, so we'll need to override HOME
				cmd := exec.Command(syncerPath, "config", "init")
				cmd.Dir = tempDir
				cmd.Env = append(os.Environ(), "HOME="+tempDir)
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				// Wait for command to complete
				Eventually(session, 10*time.Second).Should(gexec.Exit(0))

				// Verify example file was created
				examplePath := filepath.Join(tempDir, ".syncer", "config.example.yaml")
				Expect(examplePath).To(BeAnExistingFile())
			})
		})
	})
})
