package handler

import (
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"

	"github.com/registrobr/rdap-client/bootstrap"
	"github.com/registrobr/rdap-client/client"
	"github.com/registrobr/rdap-client/output"
)

var isFQDN = regexp.MustCompile(`^(([[:alnum:]](([[:alnum:]]|\-){0,61}[[:alnum:]])?\.)*[[:alnum:]](([[:alnum:]]|\-){0,61}[[:alnum:]])?)?(\.)?$`)

type Handler struct {
	URIs       []string
	HTTPClient *http.Client
	Bootstrap  *bootstrap.Client
	Writer     io.Writer
}

func (h *Handler) Query(object string) (bool, error) {
	handlers := []func(object string) (bool, error){
		h.ASN,
		h.IP,
		h.IPNetwork,
		h.Domain,
		h.Entity,
	}

	ok := false

	for _, handler := range handlers {
		var err error
		ok, err = handler(object)

		if err != nil {
			return ok, err
		}

		if ok {
			break
		}
	}

	return ok, nil
}

func (h *Handler) ASN(object string) (bool, error) {
	asn, err := strconv.ParseUint(object, 10, 32)

	if err != nil {
		return false, nil
	}

	uris := h.URIs

	if h.Bootstrap != nil {
		var err error
		uris, err = h.Bootstrap.ASN(asn)

		if err != nil {
			return true, err
		}
	}

	r, err := client.NewClient(uris, h.HTTPClient).ASN(asn)

	if err != nil {
		return true, err
	}

	as := output.AS{AS: r}
	if err := as.ToText(h.Writer); err != nil {
		return true, err
	}

	return true, nil

}

func (h *Handler) Entity(object string) (bool, error) {
	// Note that there is no bootstrap for entity, see [1]
	// [1] - https://tools.ietf.org/html/rfc7484#section-6

	r, err := client.NewClient(h.URIs, h.HTTPClient).Entity(object)
	if err != nil {
		return true, err
	}

	entity := output.Entity{Entity: r}
	if err := entity.ToText(h.Writer); err != nil {
		return true, err
	}
	return true, nil

}

func (h *Handler) IPNetwork(object string) (bool, error) {
	_, cidr, err := net.ParseCIDR(object)

	if err != nil {
		return false, nil
	}

	uris := h.URIs

	if h.Bootstrap != nil {
		var err error
		uris, err = h.Bootstrap.IPNetwork(cidr)

		if err != nil {
			return true, err
		}
	}

	r, err := client.NewClient(uris, h.HTTPClient).IPNetwork(cidr)

	if err != nil {
		return true, err
	}

	ipNetwork := output.IPNetwork{IPNetwork: r}
	if err := ipNetwork.ToText(h.Writer); err != nil {
		return true, err
	}

	return true, nil

}

func (h *Handler) IP(object string) (bool, error) {
	ip := net.ParseIP(object)

	if ip == nil {
		return false, nil
	}

	uris := h.URIs

	if h.Bootstrap != nil {
		var err error
		uris, err = h.Bootstrap.IP(ip)

		if err != nil {
			return true, err
		}
	}

	r, err := client.NewClient(uris, h.HTTPClient).IP(ip)
	if err != nil {
		return true, err
	}

	ipNetwork := output.IPNetwork{IPNetwork: r}
	if err := ipNetwork.ToText(h.Writer); err != nil {
		return true, err
	}

	return true, nil

}

func (h *Handler) Domain(object string) (bool, error) {
	if !isFQDN.MatchString(object) {
		return false, nil
	}

	uris := h.URIs

	if h.Bootstrap != nil {
		var err error
		uris, err = h.Bootstrap.Domain(object)

		if err != nil {
			return true, err
		}
	}

	r, err := client.NewClient(uris, h.HTTPClient).Domain(object)

	if err != nil {
		return true, err
	}

	if r == nil {
		return true, nil
	}

	domain := output.Domain{Domain: r}
	if err := domain.ToText(h.Writer); err != nil {
		return true, err
	}

	return true, nil
}
