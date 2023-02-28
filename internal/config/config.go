package config

import (
	"fmt"
	"strings"

	"github.com/fatih/structs"
	"github.com/spf13/viper"
)

const CONFIGS_DIR_NAME = ".config"
const PORTAL_CONFIG_DIR_NAME = "portal"
const CONFIG_FILE_NAME = "config"
const CONFIG_FILE_EXT = "yml"

type Config struct {
	Relay                string `mapstructure:"relay"`
	Verbose              bool   `mapstructure:"verbose"`
	PromptOverwriteFiles bool   `mapstructure:"prompt_overwrite_files"`
}

func GetDefault() Config {
	return Config{
		Relay:                "167.71.65.96:80",
		Verbose:              false,
		PromptOverwriteFiles: true,
	}
}

func ToMap(config Config) map[string]any {
	m := map[string]any{}
	for _, field := range structs.Fields(config) {
		key := field.Tag("mapstructure")
		value := field.Value()
		m[key] = value
	}
	return m
}

func ToYaml(config Config) []byte {
	var builder strings.Builder
	for k, v := range ToMap(config) {
		builder.WriteString(fmt.Sprintf("%s: %v", k, v))
		builder.WriteRune('\n')
	}
	return []byte(builder.String())
}

func IsDefault(key string) bool {
	defaults := ToMap(GetDefault())
	return viper.Get(key) == defaults[key]
}
