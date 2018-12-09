package registry

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/distribution/manifest/schema2"
)

// HTTPClient acts as facade on http.Client, allowing for mock implementations.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// HTTPClientImpl implements HTTPClient, and acts as a facade on http.Client, implementing just the Do method.
type HTTPClientImpl struct {
	realHTTPClient *http.Client
}

// Do sends an HTTP request and returns an HTTP response. This just calls the equivalent method on http.Client.
func (h HTTPClientImpl) Do(req *http.Request) (*http.Response, error) {
	log.Println("call to REAL http.Client.Do")
	return h.realHTTPClient.Do(req)
}

// Client represents a HTTP client connection to a Docker registry.
type Client struct {
	client       HTTPClient
	dockerConfig *configfile.ConfigFile
}

// CreateClientProvidingHTTPClient create a Client object, using a real HttpClient implementation.
func CreateClientProvidingHTTPClient(httpClient HTTPClient, dockerConfig *configfile.ConfigFile) Client {
	return Client{
		client:       httpClient,
		dockerConfig: dockerConfig,
	}
}

// CreateClient create a Client object using the provided HttpClient implementation
func CreateClient(dockerConfig *configfile.ConfigFile) Client {
	return Client{
		client: HTTPClientImpl{
			realHTTPClient: &http.Client{Timeout: 10 * time.Second},
		},
		dockerConfig: dockerConfig,
	}
}

func (c *Client) getResponse(req *http.Request, auth string) (*http.Response, error) {
	addHeaders(req, "application/vnd.docker.distribution.manifest.v2+json", auth)
	r, err := c.client.Do(req)
	if err != nil {
		log.Println("failed to get response", err)
		return nil, err
	}
	return r, nil
}

func (c *Client) getJSONFromURL(queryURL string, target interface{}) error {
	request, err := http.NewRequest("GET", queryURL, nil)
	if err != nil {
		return err
	}
	basicAuth := c.getDockerBasicAuth(request)
	response, err := c.getResponse(request, basicAuth)
	if err != nil {
		return err
	}
	switch response.StatusCode {
	case 401:
		// try bearer
		bearerAuth, err := c.getDockerBearerAuth(response, basicAuth)
		if err != nil {
			return err
		}
		response, err = c.getResponse(request, bearerAuth)
		if err != nil {
			return err
		}
		if response.StatusCode != 200 {
			return errors.New("failed to get a good response with bearer auth - status code is " + strconv.Itoa(response.StatusCode))
		}
	case 200:
		// all good - nothing to do
	default:
		// oops
		return errors.New("failed to get a good response - status code " + strconv.Itoa(response.StatusCode))
	}
	defer response.Body.Close()
	return json.NewDecoder(response.Body).Decode(target)
}

func addHeaders(req *http.Request, accept string, auth string) {
	if req.Header == nil {
		req.Header = make(map[string][]string)
	}
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	req.Header.Set("User-Agent", "regstat")
	req.Header.Set("Accept", accept)
}

// GetV2Manifest returns the Docker V2 manifest object that corresponds with the provided registry URL.
func (c *Client) GetV2Manifest(url string) (schema2.Manifest, error) {
	v2Manifest := schema2.Manifest{}
	err := c.getJSONFromURL(url, &v2Manifest)
	if err != nil {
		log.Println("failed to get v2 manifest", url, err)
	} else {
		log.Printf("successfully read v2 manifest with %d blobs\n", len(v2Manifest.Layers))
	}
	return v2Manifest, nil
}
