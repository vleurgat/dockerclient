package registry

import (
	"net/http"
	"strings"
	"testing"

	"errors"
	"io/ioutil"

	"github.com/docker/cli/cli/config/configfile"
)

func TestParseBearer(t *testing.T) {
	t.Run("no commas", func(t *testing.T) {
		kv := parseBearer("hello")
		if len(kv) != 1 || kv["hello"] != "" {
			t.Errorf("expected map of hello to empty string; got %s", kv)
		}
	})
	t.Run("three commas", func(t *testing.T) {
		kv := parseBearer("one,two,three")
		if len(kv) != 3 {
			t.Errorf("expected map with three keys; got %s", kv)
		}
		if kv["one"] != "" || kv["two"] != "" || kv["three"] != "" {
			t.Errorf("expected keys to map to empty strings; got %s", kv)
		}
	})
	t.Run("two entries", func(t *testing.T) {
		kv := parseBearer("t1=\"hello\" , t2=goodbye")
		if len(kv) != 2 {
			t.Errorf("expected map with two keys; got %s", kv)
		}
		if kv["t1"] != "hello" || kv["t2"] != "goodbye" {
			t.Errorf("expected t1:hello and t2:goodbye; got %s", kv)
		}
	})
}

func TestGetBrearerAuthURL(t *testing.T) {
	t.Run("no header", func(t *testing.T) {
		response := &http.Response{
			Header: map[string][]string{},
		}
		url, err := getBrearerAuthURL(response)
		if url != "" || err == nil {
			t.Fatalf("expected empty url and non nil error; got url %s", url)
		}
		if !strings.Contains(err.Error(), "no bearer Www-Authenticate header") {
			t.Errorf("expected no bearer Www-Authenticate header; got %s", err)
		}
	})
	t.Run("invalid url", func(t *testing.T) {
		response := &http.Response{
			Header: map[string][]string{
				"Www-Authenticate": {
					"Bearer realm=::qwertyhello",
				},
			},
		}
		url, err := getBrearerAuthURL(response)
		if url != "" || err == nil {
			t.Fatalf("expected empty url and non nil error; got url %s", url)
		}
		if !strings.Contains(err.Error(), "missing protocol") {
			t.Errorf("expected missing protocol; got %s", err)
		}
	})
	t.Run("good url", func(t *testing.T) {
		response := &http.Response{
			Header: map[string][]string{
				"Www-Authenticate": {
					"Bearer realm=http://boo,service=\"s&1\", scope=s2",
				},
			},
		}
		url, err := getBrearerAuthURL(response)
		if err != nil {
			t.Fatalf("expected nil error; got %s", err)
		}
		if url != "http://boo?service=s%261&scope=s2" && url != "http://boo?scope=s2&service=s%261" {
			t.Errorf("unexpected url; got %s", url)
		}
	})
}

func TestExtractBearerToken(t *testing.T) {
	t.Run("bad json", func(t *testing.T) {
		response := &http.Response{
			Body: ioutil.NopCloser(strings.NewReader("rubbish")),
		}
		token, err := extractBearerToken(response)
		if token != "" || err == nil {
			t.Fatalf("expected empty token and non nil error; got token %s", token)
		}
		if !strings.Contains(err.Error(), "invalid character") {
			t.Errorf("expected invalid character; got %s", err)
		}
	})
	t.Run("good token", func(t *testing.T) {
		response := &http.Response{
			Body: ioutil.NopCloser(strings.NewReader("{\"token\":\"my-token\"}")),
		}
		token, err := extractBearerToken(response)
		if err != nil {
			t.Fatalf("expected nil error; got err %s", err)
		}
		if token != "Bearer my-token" {
			t.Errorf("unexpected token; got %s", err)
		}
	})
}

func TestGetDockerBearerAuth(t *testing.T) {
	httpClient := MockHTTPClient{}
	client := Client{client: httpClient, dockerConfig: &configfile.ConfigFile{}}

	t.Run("no bearer auth url", func(t *testing.T) {
		res := &http.Response{}
		auth, err := client.getDockerBearerAuth(res, "")
		if auth != "" || err == nil {
			t.Fatalf("expected empty auth and non nil error; got auth %s", auth)
		}
		if !strings.Contains(err.Error(), "no bearer Www-Authenticate header") {
			t.Errorf("expected no bearer Www-Authenticate header; got %s", err)
		}
	})

	t.Run("invalid bearer auth url", func(t *testing.T) {
		res := &http.Response{
			Header: map[string][]string{
				"Www-Authenticate": {
					"Bearer realm=::qwertyhello",
				},
			},
		}
		auth, err := client.getDockerBearerAuth(res, "")
		if auth != "" || err == nil {
			t.Fatalf("expected empty auth and non nil error; got auth %s", auth)
		}
		if !strings.Contains(err.Error(), "missing protocol") {
			t.Errorf("expected missing protocol; got %s", err)
		}
	})

	t.Run("http Do error", func(t *testing.T) {
		httpClient := CreateMockHTTPClientErr(errors.New("oops"))
		client = Client{client: httpClient, dockerConfig: &configfile.ConfigFile{}}
		res := &http.Response{
			Header: map[string][]string{
				"Www-Authenticate": {
					"Bearer realm=http://bearer.com",
				},
			},
		}
		auth, err := client.getDockerBearerAuth(res, "")
		if auth != "" || err == nil {
			t.Fatalf("expected empty auth and non nil error; got auth %s", auth)
		}
		if !strings.Contains(err.Error(), "oops") {
			t.Errorf("expected oops; got %s", err)
		}
	})

	t.Run("http non 200", func(t *testing.T) {
		httpClient := CreateMockHTTPClient(http.Response{StatusCode: 500})
		client = Client{client: httpClient, dockerConfig: &configfile.ConfigFile{}}
		res := &http.Response{
			Header: map[string][]string{
				"Www-Authenticate": {
					"Bearer realm=http://bearer.com",
				},
			},
		}
		auth, err := client.getDockerBearerAuth(res, "")
		if auth != "" || err == nil {
			t.Fatalf("expected empty auth and non nil error; got auth %s", auth)
		}
		if !strings.Contains(err.Error(), "status code is 500") {
			t.Errorf("expected status code is 500; got %s", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		httpClient := CreateMockHTTPClient(http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader("{\"token\":\"my-token\"}")),
		})
		client = Client{client: httpClient, dockerConfig: &configfile.ConfigFile{}}
		res := &http.Response{
			Header: map[string][]string{
				"Www-Authenticate": {
					"Bearer realm=http://bearer.com",
				},
			},
		}
		auth, err := client.getDockerBearerAuth(res, "")
		if auth != "Bearer my-token" || err != nil {
			t.Fatalf("expected good auth and nil error; got auth %s; got err %s", auth, err)
		}
	})
}
