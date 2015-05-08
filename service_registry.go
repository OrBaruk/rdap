package rdap

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

// ServiceRegistry reflects the structure of a RDAP Bootstrap Service
// Registry.
//
// See http://tools.ietf.org/html/rfc7484#section-10.2
type ServiceRegistry struct {
	Version     string       `json:"version"`
	Publication time.Time    `json:"publication"`
	Description string       `json:"description,omitempty"`
	Services    ServicesList `json:"services"`
}

// ServicesList is an array of services
type ServicesList []Service

// Service is an array composed by two items. The first one is a list of
// entries and the second one is a list of URIs.
type Service [2]Values

// Values can represent either a list of entries or a list of URIs.
// It automatically sorts its URIs during the unmarshalling to prioritize
// HTTPS addresses.
type Values []string

// Entries is a helper that returns the list of entries of a service
func (s Service) Entries() []string {
	return s[0]
}

// URIs is a helper that returns the list of URIs of a service
func (s Service) URIs() []string {
	return s[1]
}

func (s *Service) UnmarshalJSON(b []byte) error {
	sv := [2]Values{}

	if err := json.Unmarshal(b, &sv); err != nil {
		return err
	}

	sort.Sort(sv[1])
	*s = sv

	return nil
}

func (v Values) Len() int {
	return len(v)
}

func (v Values) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v Values) Less(i, j int) bool {
	return strings.Split(v[i], ":")[0] == "https"
}
