package netguard

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"syscall"
	"time"
)

var ErrBlocked = errors.New("netguard: address is not public")

var ErrURL = errors.New("netguard: only http and https links without credentials are fetched")

var blocked = func() []netip.Prefix {
	var out []netip.Prefix
	for _, s := range []string{
		"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16", "172.16.0.0/12",
		"192.0.0.0/24", "192.0.2.0/24", "192.88.99.0/24", "192.168.0.0/16", "198.18.0.0/15", "198.51.100.0/24",
		"203.0.113.0/24", "224.0.0.0/4", "240.0.0.0/4",
		"::/128", "::1/128", "100::/64", "2001::/32", "2001:db8::/32", "2002::/16", "fc00::/7", "fe80::/10", "ff00::/8",
	} {
		out = append(out, netip.MustParsePrefix(s))
	}
	return out
}()

var nat64 = netip.MustParsePrefix("64:ff9b::/96")

func Blocked(a netip.Addr) bool {
	a = a.Unmap()
	if !a.IsValid() {
		return true
	}
	if nat64.Contains(a) {
		b := a.As16()
		return Blocked(netip.AddrFrom4([4]byte{b[12], b[13], b[14], b[15]}))
	}
	for _, p := range blocked {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

func CheckURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || len(raw) > 4096 {
		return nil, ErrURL
	}
	u.Fragment = ""
	return u, nil
}

type Options struct {
	AllowPrivate bool
	Timeout      time.Duration
}

func NewClient(o Options) *http.Client {
	if o.Timeout == 0 {
		o.Timeout = 15 * time.Second
	}
	d := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	if !o.AllowPrivate {
		d.Control = func(_, address string, _ syscall.RawConn) error {
			ap, err := netip.ParseAddrPort(address)
			if err != nil || Blocked(ap.Addr()) {
				return ErrBlocked
			}
			return nil
		}
	}
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, err
		}
		last := error(ErrBlocked)
		for _, a := range addrs {
			if !o.AllowPrivate && Blocked(a) {
				continue
			}
			c, err := d.DialContext(ctx, network, net.JoinHostPort(a.Unmap().String(), port))
			if err == nil {
				return c, nil
			}
			last = err
		}
		return nil, last
	}
	tr := &http.Transport{
		Proxy:                  nil,
		DialContext:            dial,
		ForceAttemptHTTP2:      true,
		MaxIdleConns:           16,
		IdleConnTimeout:        30 * time.Second,
		TLSHandshakeTimeout:    6 * time.Second,
		ResponseHeaderTimeout:  10 * time.Second,
		MaxResponseHeaderBytes: 64 << 10,
	}
	return &http.Client{
		Transport: tr,
		Timeout:   o.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("netguard: too many redirects")
			}
			if _, err := CheckURL(req.URL.String()); err != nil {
				return err
			}
			return nil
		},
	}
}

func Reason(err error) (code, message string) {
	var ne net.Error
	switch {
	case errors.Is(err, ErrBlocked):
		return "private_address", "links to private network addresses are not fetched"
	case errors.Is(err, ErrURL):
		return "bad_url", "use an http or https link"
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &ne) && ne.Timeout():
		return "fetch_timeout", "the site took too long to answer"
	}
	var de *net.DNSError
	if errors.As(err, &de) {
		return "fetch_failed", fmt.Sprintf("the host %s could not be found", de.Name)
	}
	return "fetch_failed", "the link could not be fetched"
}
