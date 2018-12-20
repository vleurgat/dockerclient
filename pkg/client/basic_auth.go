package client

import (
	"encoding/base64"
	"fmt"
)

func (c *Client) getDockerBasicAuth(host string) string {
	basicAuth := ""
	if c.dockerConfig != nil {
		config, exists := c.dockerConfig.AuthConfigs[host]
		if exists {
			if config.Auth != "" {
				basicAuth = "Basic " + config.Auth
			} else if config.Username != "" && config.Password != "" {
				basicAuth = "Basic " + base64Encode(config.Username, config.Password)
			}
		}
	}
	return basicAuth
}

func base64Encode(username string, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
}
