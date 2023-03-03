package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/structs"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

const (
	CONFIGS_DIR_NAME       = ".config"
	PORTAL_CONFIG_DIR_NAME = "portal"
	CONFIG_FILE_NAME       = "config"
	CONFIG_FILE_EXT        = "yml"

	StyleRich = "rich"
	StyleRaw  = "raw"
)

type Config struct {
	Relay                string `mapstructure:"relay"`
	Verbose              bool   `mapstructure:"verbose"`
	PromptOverwriteFiles bool   `mapstructure:"prompt_overwrite_files"`
	RelayPort            int    `mapstructure:"relay_port"`
	TuiStyle             string `mapstructure:"tui_style"`
}

func GetDefault() Config {
	return Config{
		Relay:                "167.71.65.96:80",
		Verbose:              false,
		PromptOverwriteFiles: true,
		RelayPort:            8080,
		TuiStyle:             StyleRich,
	}
}

func (config Config) Map() map[string]any {
	m := map[string]any{}
	for _, field := range structs.Fields(config) {
		key := field.Tag("mapstructure")
		value := field.Value()
		m[key] = value
	}
	return m
}

func (config Config) Yaml() []byte {
	var builder strings.Builder
	for k, v := range config.Map() {
		builder.WriteString(fmt.Sprintf("%s: %v", k, v))
		builder.WriteRune('\n')
	}
	return []byte(builder.String())
}

func IsDefault(key string) bool {
	defaults := GetDefault().Map()
	return viper.Get(key) == defaults[key]
}

// Init initializes the viper config.
// `config.yml` is created in $HOME/.config/portal if not already existing.
// NOTE: The precedence levels of viper are the following: flags -> config file -> defaults.
func Init() error {
	home, err := homedir.Dir()
	if err != nil {
		return fmt.Errorf("resolving home dir: %w", err)
	}

	configPath := filepath.Join(home, CONFIGS_DIR_NAME, PORTAL_CONFIG_DIR_NAME)
	viper.AddConfigPath(configPath)
	viper.SetConfigName(CONFIG_FILE_NAME)
	viper.SetConfigType(CONFIG_FILE_EXT)

	if err := viper.ReadInConfig(); err != nil {
		// Create config file if not found.
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			err := os.MkdirAll(configPath, os.ModePerm)
			if err != nil {
				return fmt.Errorf("Could not create config directory: %w", err)
			}

			configFile, err := os.Create(filepath.Join(configPath, fmt.Sprintf("%s.%s", CONFIG_FILE_NAME, CONFIG_FILE_EXT)))
			if err != nil {
				return fmt.Errorf("Could not create config file: %w", err)
			}
			defer configFile.Close()

			_, err = configFile.Write(GetDefault().Yaml())
			if err != nil {
				return fmt.Errorf("Could not write defaults to config file: %w", err)
			}
		} else {
			return fmt.Errorf("Could not read config file: %w", err)
		}
	}
	for k, v := range GetDefault().Map() {
		viper.SetDefault(k, v)
	}
	return nil
}
