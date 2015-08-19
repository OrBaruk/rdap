package rdap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"time"

	"strings"

	"github.com/registrobr/rdap/protocol"
)

type httpClientFunc func(*http.Request) (*http.Response, error)

func (h httpClientFunc) Do(r *http.Request) (*http.Response, error) {
	return h(r)
}

func TestDefaultFetcherFetch(t *testing.T) {
	data := []struct {
		description   string
		uris          []string
		queryType     QueryType
		queryValue    string
		xForwardedFor string
		httpClient    func() (*http.Response, error)
		expected      *http.Response
		expectedError error
	}{
		{
			description:   "it should fetch correctly",
			uris:          []string{"https://rdap.beta.registro.br"},
			queryType:     QueryTypeDomain,
			queryValue:    "example.com",
			xForwardedFor: "200.160.2.3",
			httpClient: func() (*http.Response, error) {
				domain := protocol.Domain{
					ObjectClassName: "domain",
					Handle:          "example.com",
					LDHName:         "example.com",
				}

				data, err := json.Marshal(domain)
				if err != nil {
					t.Fatal(err)
				}

				var response http.Response
				response.StatusCode = http.StatusOK
				response.Header = http.Header{
					"Content-Type": []string{"application/rdap+json"},
				}
				response.Body = nopCloser{bytes.NewBuffer(data)}
				return &response, nil
			},
			expected: func() *http.Response {
				domain := protocol.Domain{
					ObjectClassName: "domain",
					Handle:          "example.com",
					LDHName:         "example.com",
				}

				data, err := json.Marshal(domain)
				if err != nil {
					t.Fatal(err)
				}

				var response http.Response
				response.StatusCode = http.StatusOK
				response.Header = http.Header{
					"Content-Type": []string{"application/rdap+json"},
				}
				response.Body = nopCloser{bytes.NewBuffer(data)}
				return &response
			}(),
		},
		{
			description:   "it should fail to create the HTTP request",
			uris:          []string{"abc%"},
			queryType:     QueryTypeDomain,
			queryValue:    "example.com",
			expectedError: fmt.Errorf(`parse abc%%/domain/example.com: invalid URL escape "%%/d"`),
		},
		{
			description:   "it should fail while sending the HTTP request",
			uris:          []string{"https://rdap.beta.registro.br"},
			queryType:     QueryTypeDomain,
			queryValue:    "example.com",
			xForwardedFor: "200.160.2.3",
			httpClient: func() (*http.Response, error) {
				return nil, fmt.Errorf("I'm a crazy error!")
			},
			expectedError: fmt.Errorf("I'm a crazy error!"),
		},
		{
			description:   "it should store the last error (not found)",
			uris:          []string{"abc%", "https://rdap.beta.registro.br"},
			queryType:     QueryTypeDomain,
			queryValue:    "example.com",
			xForwardedFor: "200.160.2.3",
			httpClient: func() (*http.Response, error) {
				var response http.Response
				response.StatusCode = http.StatusNotFound
				return &response, nil
			},
			expectedError: ErrNotFound,
		},
		{
			description:   "it should fail when content-type isn't “application/rdap+json”",
			uris:          []string{"https://rdap.beta.registro.br"},
			queryType:     QueryTypeDomain,
			queryValue:    "example.com",
			xForwardedFor: "200.160.2.3",
			httpClient: func() (*http.Response, error) {
				var response http.Response
				response.StatusCode = http.StatusOK
				response.Header = http.Header{
					"Content-Type": []string{"text/html"},
				}
				return &response, nil
			},
			expectedError: fmt.Errorf("unexpected response: 200 OK"),
		},
		{
			description:   "it should parse an error response correctly",
			uris:          []string{"https://rdap.beta.registro.br"},
			queryType:     QueryTypeDomain,
			queryValue:    "example.com",
			xForwardedFor: "200.160.2.3",
			httpClient: func() (*http.Response, error) {
				e := protocol.Error{
					ErrorCode: 400,
				}

				data, err := json.Marshal(e)
				if err != nil {
					t.Fatal(err)
				}

				var response http.Response
				response.StatusCode = http.StatusBadRequest
				response.Header = http.Header{
					"Content-Type": []string{"application/rdap+json"},
				}
				response.Body = nopCloser{bytes.NewBuffer(data)}
				return &response, nil
			},
			expectedError: &protocol.Error{
				ErrorCode: 400,
			},
		},
		{
			description:   "it should fail to parse an invalid error response",
			uris:          []string{"https://rdap.beta.registro.br"},
			queryType:     QueryTypeDomain,
			queryValue:    "example.com",
			xForwardedFor: "200.160.2.3",
			httpClient: func() (*http.Response, error) {
				var response http.Response
				response.StatusCode = http.StatusBadRequest
				response.Header = http.Header{
					"Content-Type": []string{"application/rdap+json"},
				}
				response.Body = nopCloser{bytes.NewBufferString(`{{{`)}
				return &response, nil
			},
			expectedError: fmt.Errorf("invalid character '{' looking for beginning of object key string"),
		},
	}

	for i, item := range data {
		httpClient := httpClientFunc(func(r *http.Request) (*http.Response, error) {
			expectedURL := fmt.Sprintf("/%v/%s", item.queryType, item.queryValue)
			if r.URL.Path != expectedURL {
				return nil, fmt.Errorf("expected url “%s” and got “%s”", expectedURL, r.URL.Path)
			}

			if r.Header.Get("X-Forwarded-For") != item.xForwardedFor {
				return nil, fmt.Errorf("expected HTTP header X-Forwarded-For “%s” and got “%s”", item.xForwardedFor, r.Header.Get("X-Forwarded-For"))
			}

			return item.httpClient()
		})

		fetcher := NewDefaultFetcher(httpClient, item.xForwardedFor)
		response, err := fetcher.Fetch(item.uris, item.queryType, item.queryValue)

		if item.expectedError != nil {
			if fmt.Sprintf("%v", item.expectedError) != fmt.Sprintf("%v", err) {
				t.Errorf("[%d] %s: expected error “%v”, got “%v”", i, item.description, item.expectedError, err)
			}

		} else if err != nil {
			t.Errorf("[%d] %s: unexpected error “%s”", i, item.description, err)

		} else {
			if !reflect.DeepEqual(item.expected, response) {
				t.Errorf("[%d] “%s”: mismatch results.\n%v", i, item.description, diff(item.expected, response))
			}
		}
	}
}

func TestBootstrap(t *testing.T) {
	data := []struct {
		description   string
		uris          []string
		queryType     QueryType
		queryValue    string
		xForwardedFor string
		bootstrapURI  string
		httpClient    map[string]func() (*http.Response, error)
		cacheDetector CacheDetector
		expected      *http.Response
		expectedError error
	}{
		{
			description:   "it should retrieve the URL from bootstrap and query the RDAP server correctly",
			queryType:     QueryTypeDomain,
			queryValue:    "example.com",
			xForwardedFor: "200.160.2.3",
			bootstrapURI:  "https://data.iana.org/rdap/%s.json",
			httpClient: map[string]func() (*http.Response, error){
				"https://data.iana.org/rdap/dns.json": func() (*http.Response, error) {
					s := serviceRegistry{
						Version:     version,
						Publication: time.Now(),
						Description: "This is a test registry",
						Services: []service{
							{
								[]string{"com"},
								[]string{"https://rdap.beta.registro.br"},
							},
						},
					}

					data, err := json.Marshal(s)
					if err != nil {
						t.Fatal(err)
					}

					var response http.Response
					response.StatusCode = http.StatusOK
					response.Header = http.Header{
						"Content-Type": []string{"application/rdap+json"},
					}
					response.Body = nopCloser{bytes.NewBuffer(data)}
					return &response, nil
				},
				"https://rdap.beta.registro.br/domain/example.com": func() (*http.Response, error) {
					domain := protocol.Domain{
						ObjectClassName: "domain",
						Handle:          "example.com",
						LDHName:         "example.com",
					}

					data, err := json.Marshal(domain)
					if err != nil {
						t.Fatal(err)
					}

					var response http.Response
					response.StatusCode = http.StatusOK
					response.Header = http.Header{
						"Content-Type": []string{"application/rdap+json"},
					}
					response.Body = nopCloser{bytes.NewBuffer(data)}
					return &response, nil
				},
			},
			cacheDetector: CacheDetector(func(resp *http.Response) bool {
				return resp.Header.Get("X-From-Cache") == "1"
			}),
			expected: func() *http.Response {
				domain := protocol.Domain{
					ObjectClassName: "domain",
					Handle:          "example.com",
					LDHName:         "example.com",
				}

				data, err := json.Marshal(domain)
				if err != nil {
					t.Fatal(err)
				}

				var response http.Response
				response.StatusCode = http.StatusOK
				response.Header = http.Header{
					"Content-Type": []string{"application/rdap+json"},
				}
				response.Body = nopCloser{bytes.NewBuffer(data)}
				return &response
			}(),
		},
		{
			description:   "it should ignore entity bootstrap and query the RDAP server directly",
			uris:          []string{"https://rdap.beta.registro.br"},
			queryType:     QueryTypeEntity,
			queryValue:    "h_05506560000136-NICBR",
			xForwardedFor: "200.160.2.3",
			bootstrapURI:  "https://data.iana.org/rdap/%s.json",
			httpClient: map[string]func() (*http.Response, error){
				"https://rdap.beta.registro.br/entity/h_05506560000136-NICBR": func() (*http.Response, error) {
					entity := protocol.Entity{
						ObjectClassName: "entity",
						Handle:          "05.506.560/0001-36",
					}

					data, err := json.Marshal(entity)
					if err != nil {
						t.Fatal(err)
					}

					var response http.Response
					response.StatusCode = http.StatusOK
					response.Header = http.Header{
						"Content-Type": []string{"application/rdap+json"},
					}
					response.Body = nopCloser{bytes.NewBuffer(data)}
					return &response, nil
				},
			},
			cacheDetector: CacheDetector(func(resp *http.Response) bool {
				return resp.Header.Get("X-From-Cache") == "1"
			}),
			expected: func() *http.Response {
				entity := protocol.Entity{
					ObjectClassName: "entity",
					Handle:          "05.506.560/0001-36",
				}

				data, err := json.Marshal(entity)
				if err != nil {
					t.Fatal(err)
				}

				var response http.Response
				response.StatusCode = http.StatusOK
				response.Header = http.Header{
					"Content-Type": []string{"application/rdap+json"},
				}
				response.Body = nopCloser{bytes.NewBuffer(data)}
				return &response
			}(),
		},
		{
			description:   "it should fail to build the bootstrap request",
			queryType:     QueryTypeDomain,
			queryValue:    "example.com",
			xForwardedFor: "200.160.2.3",
			bootstrapURI:  "%sabc%",
			expectedError: fmt.Errorf(`parse dnsabc%%!(NOVERB): invalid URL escape "%%!("`),
		},
		{
			description:   "it should fail to send a bootstrap request",
			queryType:     QueryTypeDomain,
			queryValue:    "example.com",
			xForwardedFor: "200.160.2.3",
			bootstrapURI:  "https://data.iana.org/rdap/%s.json",
			httpClient: map[string]func() (*http.Response, error){
				"https://data.iana.org/rdap/dns.json": func() (*http.Response, error) {
					return nil, fmt.Errorf("I'm a crazy error!")
				},
			},
			expectedError: fmt.Errorf("I'm a crazy error!"),
		},
		{
			description:   "it should return an unexpected status from the bootstrap server",
			queryType:     QueryTypeDomain,
			queryValue:    "example.com",
			xForwardedFor: "200.160.2.3",
			bootstrapURI:  "https://data.iana.org/rdap/%s.json",
			httpClient: map[string]func() (*http.Response, error){
				"https://data.iana.org/rdap/dns.json": func() (*http.Response, error) {
					var response http.Response
					response.StatusCode = http.StatusInternalServerError
					return &response, nil
				},
			},
			expectedError: fmt.Errorf("unexpected status code 500 Internal Server Error"),
		},
		{
			description:   "it should return an invalid response from the bootstrap server",
			queryType:     QueryTypeDomain,
			queryValue:    "example.com",
			xForwardedFor: "200.160.2.3",
			bootstrapURI:  "https://data.iana.org/rdap/%s.json",
			httpClient: map[string]func() (*http.Response, error){
				"https://data.iana.org/rdap/dns.json": func() (*http.Response, error) {
					var response http.Response
					response.StatusCode = http.StatusOK
					response.Body = nopCloser{bytes.NewBufferString(`{{{{`)}
					return &response, nil
				},
			},
			expectedError: fmt.Errorf("invalid character '{' looking for beginning of object key string"),
		},
		{
			description:   "it should return an unsupported version from the bootstrap server",
			queryType:     QueryTypeDomain,
			queryValue:    "example.com",
			xForwardedFor: "200.160.2.3",
			bootstrapURI:  "https://data.iana.org/rdap/%s.json",
			httpClient: map[string]func() (*http.Response, error){
				"https://data.iana.org/rdap/dns.json": func() (*http.Response, error) {
					s := serviceRegistry{
						Version:     version + "x",
						Publication: time.Now(),
						Description: "This is a test registry",
					}

					data, err := json.Marshal(s)
					if err != nil {
						t.Fatal(err)
					}

					var response http.Response
					response.StatusCode = http.StatusOK
					response.Header = http.Header{
						"Content-Type": []string{"application/rdap+json"},
					}
					response.Body = nopCloser{bytes.NewBuffer(data)}
					return &response, nil
				},
			},
			expectedError: fmt.Errorf("incompatible bootstrap specification version: %s (expecting %s)", version+"x", version),
		},
	}

	for i, item := range data {
		httpClient := httpClientFunc(func(r *http.Request) (*http.Response, error) {
			h, ok := item.httpClient[r.URL.String()]
			if !ok {
				return nil, fmt.Errorf("no handler for URL “%s”", r.URL.String())
			}
			return h()
		})

		fetcher := NewBootstrapFetcher(httpClient, item.xForwardedFor, item.bootstrapURI, item.cacheDetector)
		response, err := fetcher.Fetch(item.uris, item.queryType, item.queryValue)

		if item.expectedError != nil {
			if fmt.Sprintf("%v", item.expectedError) != fmt.Sprintf("%v", err) {
				t.Errorf("[%d] %s: expected error “%s”, got “%s”", i, item.description, item.expectedError, err)
			}

		} else if err != nil {
			t.Errorf("[%d] %s: unexpected error “%s”", i, item.description, err)

		} else {
			if !reflect.DeepEqual(item.expected, response) {
				t.Errorf("[%d] “%s”: mismatch results.\n%v", i, item.description, diff(item.expected, response))
			}
		}
	}
}

func TestLookupNS(t *testing.T) {
	if nsSet, err := lookupNS("registro.br"); err != nil {
		t.Errorf("failed to resolve “registro.br”")

	} else {
		for _, ns := range nsSet {
			if !strings.HasSuffix(ns.Host, "dns.br.") {
				t.Errorf("unexpected host “%s” returned for registro.br", ns.Host)
			}
		}
	}

	if _, err := lookupNS("1.com.br"); err == nil {
		t.Errorf("expected an error to resolve “1.com.br”")
	}
}
