package registry

import (
	"net/http"
)

// MockHTTPClient is a mock implementation of HttpClient.
type MockHTTPClient struct {
	responses *[]http.Response
	err       error
}

// CreateMockHTTPClientErr creates a MockHttpClient that returns errors.
func CreateMockHTTPClientErr(err error) MockHTTPClient {
	return MockHTTPClient{
		responses: &[]http.Response{{}},
		err:       err,
	}
}

// CreateMockHTTPClient creates a MockHttpClient that returns http.Responses.
func CreateMockHTTPClient(res ...http.Response) MockHTTPClient {
	return MockHTTPClient{
		responses: &res,
	}
}

// Do is the mock implementation of the real http.Client.Do method.
func (m MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	response := (*m.responses)[0]
	if len(*m.responses) > 1 {
		*m.responses = (*m.responses)[1:]
	}
	return &response, m.err
}
