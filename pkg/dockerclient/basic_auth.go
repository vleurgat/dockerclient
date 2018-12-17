package dockerclient

func (c *Client) getDockerBasicAuth(host string) string {
	basicAuth := ""
	if c.dockerConfig != nil {
		config, exists := c.dockerConfig.AuthConfigs[host]
		if exists && config.Auth != "" {
			basicAuth = "Basic " + config.Auth
		}
	}
	return basicAuth
}
