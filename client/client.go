package client

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sort"

	"github.com/registrobr/rdap-client/bootstrap"
	"github.com/registrobr/rdap-client/protocol"

	"github.com/registrobr/rdap-client/Godeps/_workspace/src/github.com/gregjones/httpcache"
	"github.com/registrobr/rdap-client/Godeps/_workspace/src/github.com/gregjones/httpcache/diskcache"
)

const (
	RDAPBootstrap = "https://data.iana.org/rdap/%s.json"
)

type kind string

const (
	dns  kind = "dns"
	asn  kind = "asn"
	ipv4 kind = "ipv4"
	ipv6 kind = "ipv6"
)

var (
	kindToSegment = map[kind]string{
		dns:  "domain",
		asn:  "autnum",
		ipv4: "ip",
		ipv6: "ip",
	}
)

type Client struct {
	cacheDir  string
	Bootstrap string
	Host      string
}

func NewClient(cacheDir string) *Client {
	return &Client{
		cacheDir:  cacheDir,
		Bootstrap: RDAPBootstrap,
	}
}

func (c *Client) QueryDomain(fqdn string) (*protocol.DomainResponse, error) {
	r := &protocol.DomainResponse{}

	if err := c.query(dns, fqdn, r); err != nil {
		return nil, err
	}

	return r, nil
}

func (c *Client) QueryASN(as uint64) (*protocol.ASResponse, error) {
	r := &protocol.ASResponse{}

	if err := c.query(asn, as, r); err != nil {
		return nil, err
	}

	return r, nil
}

func (c *Client) QueryIPNetwork(ipnet *net.IPNet) (*protocol.IPNetwork, error) {
	r := &protocol.IPNetwork{}

	kind := ipv4

	if ipnet.IP.To4() == nil {
		kind = ipv6
	}

	if err := c.query(kind, ipnet, r); err != nil {
		return nil, err
	}

	return r, nil
}

func (c *Client) query(kind kind, identifier interface{}, object interface{}) error {
	uris := []string{}
	r := bootstrap.ServiceRegistry{}
	if c.Host == "" {
		bootstrapURI := fmt.Sprintf(c.Bootstrap, kind)
		if err := c.fetchAndUnmarshal(bootstrapURI, &r); err != nil {
			return err
		}

		var err error
		switch kind {
		case dns:
			uris, err = r.MatchDomain(identifier.(string))
		case asn:
			uris, err = r.MatchAS(identifier.(uint64))
		case ipv4, ipv6:
			uris, err = r.MatchIPNetwork(identifier.(*net.IPNet))
		}

		if err != nil {
			return err
		}

		if len(uris) == 0 {
			return fmt.Errorf("no matches for %v", identifier)
		}
	} else {
		uris = []string{c.Host}
	}

	sort.Sort(bootstrap.Values(uris))
	segment := kindToSegment[kind]

	for _, uri := range uris {
		err := c.fetchAndUnmarshal(fmt.Sprintf("%s/%s/%v", uri, segment, identifier), object)
		if err != nil {
			continue
		}

		return nil
	}

	return fmt.Errorf("no data available for %v", identifier)
}

func (c *Client) fetchAndUnmarshal(uri string, object interface{}) error {
	cli := http.Client{
		Transport: httpcache.NewTransport(
			diskcache.New(c.cacheDir),
		),
	}

	req, err := http.NewRequest("GET", uri, nil)

	if err != nil {
		return err
	}

	resp, err := cli.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&object); err != nil {
		return err
	}

	return nil
}
