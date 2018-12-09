package registry

import (
	"errors"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
)

type testStruct struct {
	Name string `json:"name"`
}

func TestGetResponse(t *testing.T) {
	t.Run("failure", func(t *testing.T) {
		httpClient := CreateMockHTTPClientErr(errors.New("oops"))
		client := Client{client: httpClient, dockerConfig: &configfile.ConfigFile{}}
		req := &http.Request{}
		res, err := client.getResponse(req, "auth")
		if res != nil || err == nil {
			t.Fatal("expected nil response and non nil error", res)
		}
		if !strings.Contains(err.Error(), "oops") {
			t.Errorf("expected oops; got %s", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		expectedResponse := http.Response{StatusCode: 200}
		httpClient := CreateMockHTTPClient(expectedResponse)
		client := Client{client: httpClient, dockerConfig: &configfile.ConfigFile{}}
		req := &http.Request{}
		res, err := client.getResponse(req, "auth")
		if !reflect.DeepEqual(*res, expectedResponse) || err != nil {
			t.Error("expected matching response and nil error")
		}
		if req.Header.Get("Authorization") != "auth" {
			t.Error("expected headers to be updated: Authorization")
		}
		if req.Header.Get("User-Agent") != "regstat" {
			t.Error("expected headers to be updated: User-Agent")
		}
		if req.Header.Get("Accept") != "application/vnd.docker.distribution.manifest.v2+json" {
			t.Error("expected headers to be updated: Accept")
		}
	})
}

func TestGetJSONFromURL(t *testing.T) {
	t.Run("bad request", func(t *testing.T) {
		client := Client{client: nil, dockerConfig: nil}
		testObject := testStruct{}
		err := client.getJSONFromURL("::qwertyhello", &testObject)
		if err == nil {
			t.Fatal("expected error to be non nil")
		}
		if !strings.Contains(err.Error(), "missing protocol") {
			t.Errorf("expected missing protocol; got %s", err)
		}
	})

	t.Run("bad basic auth", func(t *testing.T) {
		client := Client{
			client:       CreateMockHTTPClientErr(errors.New("boo")),
			dockerConfig: nil,
		}
		testObject := testStruct{}
		err := client.getJSONFromURL("http://hello", &testObject)
		if err == nil {
			t.Fatal("expected error to be non nil")
		}
		if !strings.Contains(err.Error(), "boo") {
			t.Errorf("expected boo; got %s", err)
		}
	})

	t.Run("non-200, non-401 basic auth", func(t *testing.T) {
		client := Client{
			client:       CreateMockHTTPClient(http.Response{StatusCode: 500}),
			dockerConfig: nil,
		}
		testObject := testStruct{}
		err := client.getJSONFromURL("http://hello", &testObject)
		if err == nil {
			t.Fatal("expected error to be non nil")
		}
		if !strings.Contains(err.Error(), "status code 500") {
			t.Errorf("expected status code 500; got %s", err)
		}
	})

	t.Run("200 basic auth", func(t *testing.T) {
		client := Client{
			client: CreateMockHTTPClient(http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader("{\"name\":\"hello\"}")),
			}),
			dockerConfig: nil}
		testObject := testStruct{}
		err := client.getJSONFromURL("http://hello", &testObject)
		if err != nil {
			t.Fatal("expected error to be nil")
		}
		if testObject.Name != "hello" {
			t.Errorf("unexpected response body; got %s", testObject)
		}
	})

	t.Run("401 bearer auth error", func(t *testing.T) {
		client := Client{
			client:       CreateMockHTTPClient(http.Response{StatusCode: 401}),
			dockerConfig: nil}
		testObject := testStruct{}
		err := client.getJSONFromURL("http://hello", &testObject)
		if err == nil {
			t.Fatal("expected error to be non nil")
		}
		if !strings.Contains(err.Error(), "no bearer Www-Authenticate header") {
			t.Errorf("expected no bearer Www-Authenticate header; got %s", err)
		}
	})

	t.Run("401 bearer bad url", func(t *testing.T) {
		client := Client{
			client: CreateMockHTTPClient(http.Response{
				StatusCode: 401,
				Header: map[string][]string{
					"Www-Authenticate": {
						"Bearer realm=::qwertyhello",
					},
				},
			}),
			dockerConfig: nil,
		}
		testObject := testStruct{}
		err := client.getJSONFromURL("http://hello", &testObject)
		if err == nil {
			t.Fatal("expected error to be non nil")
		}
		if !strings.Contains(err.Error(), "missing protocol") {
			t.Errorf("expected missing protocol; got %s", err)
		}
	})

	t.Run("500 from bearer token", func(t *testing.T) {
		client := Client{
			client: CreateMockHTTPClient(
				http.Response{
					StatusCode: 401,
					Header: map[string][]string{
						"Www-Authenticate": {
							"Bearer realm=http://bearer",
						},
					},
				}, http.Response{
					StatusCode: 500,
				},
			),
			dockerConfig: nil,
		}
		testObject := testStruct{}
		err := client.getJSONFromURL("http://hello", &testObject)
		if err == nil {
			t.Fatal("expected error to be non nil")
		}
		if !strings.Contains(err.Error(), "status code is 500") {
			t.Errorf("expected status code is 500; got %s", err)
		}
	})

	t.Run("bearer token success; 500 from auth", func(t *testing.T) {
		client := Client{
			client: CreateMockHTTPClient(
				http.Response{
					StatusCode: 401,
					Header: map[string][]string{
						"Www-Authenticate": {
							"Bearer realm=http://bearer",
						},
					},
				}, http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader("{\"token\":\"my-token\"}")),
				}, http.Response{
					StatusCode: 500,
				},
			),
			dockerConfig: nil,
		}
		testObject := testStruct{}
		err := client.getJSONFromURL("http://hello", &testObject)
		if err == nil {
			t.Fatal("expected error to be non nil")
		}
		if !strings.Contains(err.Error(), "status code is 500") {
			t.Errorf("expected status code is 500; got %s", err)
		}
	})

	t.Run("bearer auth success", func(t *testing.T) {
		client := Client{
			client: CreateMockHTTPClient(
				http.Response{
					StatusCode: 401,
					Header: map[string][]string{
						"Www-Authenticate": {
							"Bearer realm=http://bearer",
						},
					},
				}, http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader("{\"token\":\"my-token\"}")),
				}, http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader("{\"name\":\"hello\"}")),
				},
			),
			dockerConfig: nil,
		}
		testObject := testStruct{}
		err := client.getJSONFromURL("http://hello", &testObject)
		if err != nil {
			t.Fatal("expected error to be nil", err)
		}
		if testObject.Name != "hello" {
			t.Errorf("unexpected response body; got %s", testObject)
		}
	})

}
