package client

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

func parseBearer(bearer string) map[string]string {
	kv := make(map[string]string)
	// we're processing a suffix like: foo="hello,world",bar="abc",goo="anything"
	// which should result in the map: {foo:hello,world bar:abc goo:anything}
	rx := regexp.MustCompile("[a-zA-Z0-9]+=\"[^\"]+\"")
	tokens := rx.FindAllString(bearer, -1)
	for _, token := range tokens {
		token = strings.Trim(token, " ")
		if parts := strings.SplitN(token, "=", 2); len(parts) == 2 {
			kv[parts[0]] = strings.Trim(parts[1], `"`)
		} else {
			kv[token] = ""
		}
	}
	return kv
}

func getBearerAuthURL(response *http.Response) (string, error) {
	header := response.Header.Get("Www-Authenticate")
	if !strings.HasPrefix(header, "Bearer ") {
		return "", errors.New("no bearer Www-Authenticate header")
	}
	bearer := header[7:]
	bearerKv := parseBearer(bearer)
	bearerURL, err := url.Parse(bearerKv["realm"])
	if err != nil {
		return "", err
	}
	bearerURL.RawQuery = url.Values{
		"service": []string{bearerKv["service"]},
		"scope":   []string{bearerKv["scope"]},
	}.Encode()
	return bearerURL.String(), nil
}

func extractBearerToken(response *http.Response) (string, error) {
	defer response.Body.Close()
	type tokenResponse struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	tr := new(tokenResponse)
	err := json.NewDecoder(response.Body).Decode(tr)
	if err != nil {
		return "", err
	}
	return "Bearer " + tr.Token, nil
}

func (c *Client) getDockerBearerAuth(response *http.Response, basicAuth string) (string, error) {
	bearerURL, err := getBearerAuthURL(response)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("GET", bearerURL, nil)
	if err != nil {
		return "", err
	}
	setHeader(req, "Authorization", basicAuth)
	setHeader(req, "Accept", "application/json")
	response, err = c.client.Do(req)
	if err != nil {
		return "", err
	}
	if response.StatusCode != 200 {
		return "", errors.New("failed to determine the bearer token - status code is " + strconv.Itoa(response.StatusCode))
	}
	return extractBearerToken(response)
}
