package syncer

import (
	"fmt"
	"os"
	"path/filepath"
	"unicode"

	"github.com/TrevorEdris/retropie-utils/pkg/errors"
	"github.com/TrevorEdris/retropie-utils/pkg/storage"
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

type (
	// TODO: Allow for arbitrary locations?
	Config struct {
		Username   string  `mapstructure:"username" validate:"required"`
		Storage    Storage `mapstructure:"storage"`
		RomsFolder string  `mapstructure:"romsFolder"`
		Sync       Sync    `mapstructure:"sync"`
	}

	Storage struct {
		S3 storage.S3Config `mapstructure:"s3"`
	}

	Sync struct {
		Roms   bool `mapstructure:"roms"`
		Saves  bool `mapstructure:"saves"`
		States bool `mapstructure:"states"`
	}
)

const (
	DefaultUsername = "DEFAULT_USERNAME_CHANGE_THIS_VALUE"

	UsernameMinLength = 3
	UsernameMaxLength = 1024
)

var (
	additionalUsernameChars = []rune{'-', '_', '.'}
)

var example = Config{
	Username: DefaultUsername,
	Storage: Storage{
		S3: storage.S3Config{
			Enabled: true,
			Bucket:  "retropie-sync",
		},
	},
	Sync: Sync{
		Roms:   false,
		Saves:  true,
		States: true,
	},
}

var validate *validator.Validate

func init() {
	validate = validator.New()
}

func CreateExample(outputDir string) error {
	err := os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		return err
	}
	filename := filepath.Join(outputDir, "config.example.yaml")
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			panic(err)
		}
	}(f)
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	example.RomsFolder = filepath.Join(userHomeDir, "RetroPie", "roms")
	yamlData, err := yaml.Marshal(&example)
	if err != nil {
		return err
	}
	_, err = f.Write(yamlData)
	if err != nil {
		return err
	}
	fmt.Printf("Created %s\n", filename)
	return nil
}

func ValidateConfigFile(configFile string) error {
	bytes, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}
	config := &Config{}
	err = yaml.Unmarshal(bytes, config)
	if err != nil {
		return err
	}

	return ValidateConfig(config)
}

func ValidateConfig(config *Config) error {
	err := validate.Struct(config)
	if err != nil {
		return err
	}

	err = validateUsername(config.Username)
	if err != nil {
		return err
	}

	return nil
}

func validateUsername(username string) error {
	if username == DefaultUsername {
		return errors.DefaultUsernameError
	}

	if username == "" {
		return errors.NewInvalidUsernameWithReasonError("username is empty")
	}

	err := validateContainsOnlySupportedChars(username)
	if err != nil {
		return err
	}

	return nil
}

func validateContainsOnlySupportedChars(username string) error {
	if len(username) < UsernameMinLength || len(username) > UsernameMaxLength {
		return errors.NewInvalidUsernameWithReasonError(fmt.Sprintf("username has invalid length; Must be %d <= length <= %d", UsernameMinLength, UsernameMaxLength))
	}

	for _, c := range username {
		if unicode.IsLetter(c) || unicode.IsNumber(c) {
			continue
		}
		isSupported := false
		for _, r := range additionalUsernameChars {
			if c == r {
				isSupported = true
			}
		}
		if !isSupported {
			return errors.NewInvalidUsernameWithReasonError(fmt.Sprintf("username contains illegal character '%c'; Must be alphanumeric", c))
		}
	}

	return nil
}
