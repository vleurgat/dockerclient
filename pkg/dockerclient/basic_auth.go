package dockerclient

import (
	"net/http"
)

func (c *Client) getDockerBasicAuth(req *http.Request) string {
	basicAuth := ""
	if c.dockerConfig != nil {
		config, exists := c.dockerConfig.AuthConfigs[req.Host]
		if exists && config.Auth != "" {
			basicAuth = "Basic " + config.Auth
		}
	}
	return basicAuth
}
