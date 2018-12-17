package dockerclient

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
)

func TestGetDockerBasicAuth(t *testing.T) {
	client := Client{client: nil, dockerConfig: &configfile.ConfigFile{}}
	var req http.Request
	req.Host = "my.host"

	t.Run("no auth", func(t *testing.T) {
		client = Client{client: nil, dockerConfig: nil}
		auth := client.getDockerBasicAuth("my.host")
		if auth != "" {
			t.Errorf("expected empty auth string; got %s", auth)
		}
	})

	t.Run("no match", func(t *testing.T) {
		auth := client.getDockerBasicAuth("my.host")
		if auth != "" {
			t.Errorf("expected empty auth string; got %s", auth)
		}
	})

	t.Run("success", func(t *testing.T) {
		configFile := &configfile.ConfigFile{}
		err := json.NewDecoder(strings.NewReader("{\"auths\":{\"my.host\": {\"auth\":\"token\"}}}")).Decode(configFile)
		if err != nil {
			t.Error("failed to read JSON", err)
		}
		client.dockerConfig = configFile
		auth := client.getDockerBasicAuth("my.host")
		if auth != "Basic token" {
			t.Errorf("expected 'Basic token'; got %s", auth)
		}
	})
}
