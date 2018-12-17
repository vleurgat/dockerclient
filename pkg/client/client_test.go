package client

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

type testStruct struct {
	Name string `json:"name"`
}

func TestDoGet(t *testing.T) {
	t.Run("bad request", func(t *testing.T) {
		client := Client{client: nil, dockerConfig: nil}
		testObject := testStruct{}
		err := client.doGet("::qwertyhello", &testObject)
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
		err := client.doGet("http://hello", &testObject)
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
		err := client.doGet("http://hello", &testObject)
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
		err := client.doGet("http://hello", &testObject)
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
		err := client.doGet("http://hello", &testObject)
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
		err := client.doGet("http://hello", &testObject)
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
		err := client.doGet("http://hello", &testObject)
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
		err := client.doGet("http://hello", &testObject)
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
		err := client.doGet("http://hello", &testObject)
		if err != nil {
			t.Fatal("expected error to be nil", err)
		}
		if testObject.Name != "hello" {
			t.Errorf("unexpected response body; got %s", testObject)
		}
	})

}
