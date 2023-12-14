package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/TrevorEdris/retropie-utils/pkg/storage"
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

type (
	Config struct {
		Emulators []string
		FileTypes FileTypes
		Storage   Storage
	}

	FileTypes struct {
		Roms   bool
		Saves  bool
		States bool
	}

	Storage struct {
		GoogleDrive storage.GDriveConfig
		S3          storage.S3Config
		SFTP        storage.SFTPConfig
	}
)

var example = Config{
	Emulators: []string{"gb", "gbc", "gba", "snes", "n64"},
	FileTypes: FileTypes{
		Roms:   false,
		Saves:  true,
		States: true,
	},
	Storage: Storage{
		GoogleDrive: storage.GDriveConfig{
			Enabled: false,
		},
		S3: storage.S3Config{
			Enabled: true,
			Bucket:  "retropie-sync",
		},
		SFTP: storage.SFTPConfig{
			Enabled: false,
		},
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
