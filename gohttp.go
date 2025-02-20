package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ProxyAddr holds the proxy address.
var ProxyAddr string

var transport = &http.Transport{
	TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
	DisableKeepAlives: true,
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: time.Second,
		DualStack: true,
	}).DialContext,
	Proxy: func(request *http.Request) (*url.URL, error) {
		ProxyAddr = "http://172.22.112.1:8081"
		if ProxyAddr != "" {
			return url.Parse(ProxyAddr)
		}
		return nil, nil
	},
}

var httpClient = &http.Client{
	Transport: transport,
}

func goRequest(r request) response {
	httpClient.Timeout = r.timeout

	if !r.followLocation {
		httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	var req *http.Request
	var err error
	if r.body != "" {
		req, err = http.NewRequest(r.method, r.URL(), bytes.NewBuffer([]byte(r.body)))
	} else {
		req, err = http.NewRequest(r.method, r.URL(), nil)
	}

	if err != nil {
		return response{request: r, err: err}
	}
	req.Close = true

	if !r.HasHeader("Host") {
		// add the host header to the request manually so it shows up in the output
		r.headers = append(r.headers, fmt.Sprintf("Host: %s", r.Hostname()))
	}

	if !r.HasHeader("User-Agent") {
		r.headers = append(r.headers, fmt.Sprintf("User-Agent: %s", userAgent))
	}

	for _, h := range r.headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) != 2 {
			continue
		}

		req.Header.Set(parts[0], parts[1])
	}

	resp, err := httpClient.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return response{request: r, err: err}
	}
	body, _ := ioutil.ReadAll(resp.Body)

	// extract the response headers
	hs := make([]string, 0)
	for k, vs := range resp.Header {
		for _, v := range vs {
			hs = append(hs, fmt.Sprintf("%s: %s", k, v))
		}
	}

	return response{
		request:    r,
		status:     resp.Status,
		statusCode: resp.StatusCode,
		headers:    hs,
		body:       body,
	}
}

