package ping

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/hellodword/cfping/bind"
)

type Data struct {
	IP    string
	Delay int64
}

const (
	// UserAgent default user-agent
	UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.102 Safari/537.36"
)

func Cloudflare(url, ip, iFace string, status int, timeout int) (*Data, error) {
	var err error

	var dialer *net.Dialer

	if iFace == "" {
		dialer = &net.Dialer{}
	} else {
		dialer, err = bind.NewTCPDialerFromInterface(iFace)
		if err != nil {
			return nil, err
		}
	}

	dialer.Timeout = time.Millisecond * time.Duration(timeout)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
			MaxVersion: tls.VersionTLS13,
		},
		ForceAttemptHTTP2: false,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if strings.Index(addr, ":443") != -1 {
				addr = fmt.Sprintf("%s:443", ip)
				return dialer.DialContext(ctx, network, addr)
			}
			return nil, fmt.Errorf("not https")
		},

		TLSHandshakeTimeout:   dialer.Timeout,
		ExpectContinueTimeout: dialer.Timeout,
		IdleConnTimeout:       dialer.Timeout,
		ResponseHeaderTimeout: dialer.Timeout,
	}
	c := &http.Client{
		Transport: tr,
		Timeout:   dialer.Timeout * 2,
	}

	request, err := http.NewRequest(http.MethodGet, url, nil)
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

	//defer response.Body.Close()
	//body, _ := ioutil.ReadAll(response.Body)
	//
	//arr := strings.Split(string(body), "\n")
	//for i := range arr {
	//	arr2 := strings.Split(arr[i], "=")
	//	if len(arr2) == 2 && arr2[0] == "IP" {
	//		ts := strings.ReplaceAll(arr2[1], ".", "")
	//
	//		return &PingData{
	//			IP:    IP,
	//			Delay: 0,
	//		}, nil
	//	}
	//}
	//
	//return nil, errors.New(string(body))
}
