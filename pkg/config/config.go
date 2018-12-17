package config

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/docker/cli/cli/config/configfile"
)

// CreateConfig creates a Docker config file object.
func CreateConfig(dockerConfigFile string) (*configfile.ConfigFile, error) {
	var configFile *configfile.ConfigFile
	var err error
	if dockerConfigFile != "" {
		var file *os.File
		file, err = os.Open(dockerConfigFile)
		if err == nil {
			configFile = &configfile.ConfigFile{}
			err = json.NewDecoder(bufio.NewReader(file)).Decode(configFile)
		}
	}
	if err != nil {
		configFile = nil
	}
	return configFile, err
}
