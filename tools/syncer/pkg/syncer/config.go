package syncer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/TrevorEdris/retropie-utils/pkg/storage"
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

type (
	// TODO: Allow for arbitrary locations?
	Config struct {
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

var example = Config{
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
	defer f.Close()
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

func ValidateConfig(configFile string) error {
	validate = validator.New()

	bytes, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}
	config := &Config{}
	err = yaml.Unmarshal(bytes, config)
	if err != nil {
		return err
	}

	err = validate.Struct(config)
	if err != nil {
		return err
	}
	return nil
}
