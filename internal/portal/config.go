//nolint:errcheck
package portal

import (
	"bytes"
	"encoding/json"
	"io"
)

// defaultConfig specifies the default config for the portal module.
var defaultConfig = Config{
	RendezvousAddr: "portal.spatiumportae.com",
}

// Config specifes a config for the portal module.
type Config struct {
	RendezvousAddr string `json:"RendezvousAddr,omitempty"`
}

// MergeConfigReader merges the config from the reader
// with into the provided config. Values in the reader
// will override values in the provided config
func MergeConfigReader(dst Config, r io.Reader) Config {
	json.NewDecoder(r).Decode(&dst)
	return dst
}

// MergeConfig merges the specified source config into the
// specified destination config. Values present in the source
// config will overide values in the destination config.
func MergeConfig(dst Config, src *Config) Config {
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(src)
	return MergeConfigReader(dst, &buf)
}
