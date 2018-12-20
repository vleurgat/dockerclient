package client

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
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
	return h.realHTTPClient.Do(req)
}

// Client represents a HTTP client connection to a Docker registry.
type Client struct {
	client       HTTPClient
	dockerConfig *configfile.ConfigFile
}

// CreateClientProvidingHTTPClient create a Client object, using the provided HttpClient implementation.
func CreateClientProvidingHTTPClient(httpClient HTTPClient, dockerConfig *configfile.ConfigFile) Client {
	return Client{
		client:       httpClient,
		dockerConfig: dockerConfig,
	}
}

// CreateClient create a Client object, using a real HttpClient implementation.
func CreateClient(dockerConfig *configfile.ConfigFile) Client {
	return Client{
		client: HTTPClientImpl{
			realHTTPClient: &http.Client{Timeout: 10 * time.Second},
		},
		dockerConfig: dockerConfig,
	}
}

func (c *Client) doGet(queryURL string, target interface{}) error {
	request, err := http.NewRequest("GET", queryURL, nil)
	if err != nil {
		return err
	}
	setHeader(request, "Accept", "application/vnd.docker.distribution.manifest.v2+json")
	return c.doRequest(request, target, "")
}

func (c *Client) doPut(queryURL string, payload interface{}) error {
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	body := string(jsonBody)
	request, err := http.NewRequest("PUT", queryURL, nil)
	if err != nil {
		return err
	}
	setBody(request, body)
	setHeader(request, "Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
	return c.doRequest(request, nil, body)
}

func setHeader(request *http.Request, header string, value string) {
	if header != "" && value != "" {
		request.Header.Set(header, value)
	}
}

func setBody(request *http.Request, body string) {
	if body != "" {
		request.Body = ioutil.NopCloser(strings.NewReader(body))
	}
}

func (c *Client) doRequest(request *http.Request, target interface{}, body string) error {
	basicAuth := c.getDockerBasicAuth(request.Host)
	setHeader(request, "Authorization", basicAuth)
	response, err := c.client.Do(request)
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
		setHeader(request, "Authorization", bearerAuth)
		setBody(request, body)
		response, err = c.client.Do(request)
		if err != nil {
			return err
		}
		if response.StatusCode != 200 && response.StatusCode != 201 {
			return errors.New("failed to get a good response with bearer auth - status code is " + strconv.Itoa(response.StatusCode))
		}
	case 200, 201:
		// all good - nothing to do
	default:
		// oops
		return errors.New("failed to get a good response - status code " + strconv.Itoa(response.StatusCode))
	}
	err = nil
	if target != nil {
		defer response.Body.Close()
		err = json.NewDecoder(response.Body).Decode(target)
	}
	return err
}

// GetV2Manifest returns the Docker V2 manifest object that corresponds with the provided registry URL.
func (c *Client) GetV2Manifest(url string) (schema2.Manifest, error) {
	v2Manifest := schema2.Manifest{}
	err := c.doGet(url, &v2Manifest)
	if err != nil {
		log.Println("failed to GET v2 manifest", url, err)
	}
	return v2Manifest, err
}

// PutV2Manifest associates the Docker V2 manifest object with the given tag.
func (c *Client) PutV2Manifest(url string, v2Manifest schema2.Manifest) error {
	err := c.doPut(url, v2Manifest)
	if err != nil {
		log.Println("failed to PUT v2 manifest", url, err)
	}
	return err
}
