package ping

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/hellodword/cfping/utils/bind"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// UserAgent default user-agent
	UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.102 Safari/537.36"
)

type Data struct {
	IP    string
	Delay int64
}

type SortedData []*Data

func (e SortedData) Len() int {
	return len(e)
}

func (e SortedData) Less(i, j int) bool {
	return e[i].Delay < e[j].Delay
}

func (e SortedData) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (e SortedData) Remove(i int) SortedData {
	return append(e[:i], e[i+1:]...)
}

func Cloudflare(link, ip, iFace string, status int, timeout int, http2 bool, minTls int, insecure bool, proxyStr string) (*Data, error) {
	var err error

	var dialer *net.Dialer
	var dialFunc func(context.Context, string, string) (net.Conn, error)

	if iFace == "" {
		dialer = &net.Dialer{}
	} else {
		dialer, err = bind.NewTCPDialerFromInterface(iFace)
		if err != nil {
			return nil, err
		}
	}

	dialer.Timeout = time.Millisecond * time.Duration(timeout)

	if proxyStr != "" {
		u, err := url.Parse(proxyStr)
		if err != nil {
			return nil, err
		}
		p, err := proxy.FromURL(u, dialer)
		if err != nil {
			return nil, err
		}

		dialFunc = func(ctx context.Context, network string, addr string) (net.Conn, error) {
			return p.Dial(network, addr)
		}
	} else {
		dialFunc = dialer.DialContext
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:         uint16(minTls + tls.VersionTLS10),
			MaxVersion:         tls.VersionTLS13,
			InsecureSkipVerify: insecure,
		},
		ForceAttemptHTTP2: http2,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if strings.Index(addr, ":443") != -1 {
				addr = fmt.Sprintf("%s:443", ip)
				return dialFunc(ctx, network, addr)
			}
			return nil, fmt.Errorf("not https")
		},

		TLSHandshakeTimeout:   time.Millisecond * time.Duration(timeout),
		ExpectContinueTimeout: time.Millisecond * time.Duration(timeout),
		IdleConnTimeout:       time.Millisecond * time.Duration(timeout),
		ResponseHeaderTimeout: time.Millisecond * time.Duration(timeout),
	}
	c := &http.Client{
		Transport: tr,
		Timeout:   time.Millisecond * time.Duration(timeout) * 2,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	request, err := http.NewRequest(http.MethodGet, link, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", UserAgent)

	ts := time.Now()
	response, err := c.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != status {
		err = fmt.Errorf("expected status code %d but got %d", status, response.StatusCode)
		return nil, err
	}
	delay := time.Since(ts)

	return &Data{
		IP:    ip,
		Delay: delay.Milliseconds(),
	}, nil
}
