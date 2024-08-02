package syncer_test

import (
	"github.com/TrevorEdris/retropie-utils/pkg/errors"
	"github.com/TrevorEdris/retropie-utils/pkg/storage"
	"github.com/TrevorEdris/retropie-utils/tools/syncer/pkg/syncer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	It("compiles", func() {
		Expect(nil).To(BeNil())
	})

	validConfig := syncer.Config{
		Username: "asdf1234",
		Storage: syncer.Storage{
			S3: storage.S3Config{
				Enabled: true,
				Bucket:  "test-bucket",
			},
		},
		Sync: syncer.Sync{
			Roms:   false,
			Saves:  true,
			States: true,
		},
	}

	Context("config is valid", func() {
		It("passes validation", func() {
			err := syncer.ValidateConfig(&validConfig)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("config is invalid", func() {
		When("username is too short", func() {
			It("fails with invalid length error", func() {
				cfg := validConfig
				cfg.Username = "a"

				err := syncer.ValidateConfig(&cfg)
				Expect(err).To(MatchError(errors.InvalidUsernameError))
				Expect(err.Error()).To(ContainSubstring("username has invalid length"))
			})
		})

		When("username is too long", func() {
			It("fails with invalid length error", func() {
				cfg := validConfig
				s := ""
				for i := 0; i < syncer.UsernameMaxLength+1; i++ {
					s += "a"
				}
				cfg.Username = s

				err := syncer.ValidateConfig(&cfg)
				Expect(err).To(MatchError(errors.InvalidUsernameError))
				Expect(err.Error()).To(ContainSubstring("username has invalid length"))
			})
		})

		When("username contains invalid characters", func() {
			It("fails with unsupported characters error", func() {
				cfg := validConfig
				cfg.Username = "look/at/meI'm%special"

				err := syncer.ValidateConfig(&cfg)
				Expect(err).To(MatchError(errors.InvalidUsernameError))
				Expect(err.Error()).To(ContainSubstring("username contains illegal character"))
			})
		})

		When("username is default username", func() {
			It("fails with default username error", func() {
				cfg := validConfig
				cfg.Username = syncer.DefaultUsername

				err := syncer.ValidateConfig(&cfg)
				Expect(err).To(MatchError(errors.DefaultUsernameError))
			})
		})
	})
})
