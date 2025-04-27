package httpclient

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HttpClient struct {
	signCertificate SignCertificate
	header          map[string]string
	httpCli         *http.Client
	query           map[string]string
	timeout         time.Duration // http请求超时
}

func NewHttpClient(signCertificate SignCertificate) *HttpClient {
	return &HttpClient{
		signCertificate: signCertificate,
		httpCli:         &http.Client{},
		header:          make(map[string]string),
		query:           make(map[string]string),
		timeout:         0, // 默认超时时间为0，表示没有设置超时
	}
}

// SetTimeout 设置超时时间
func (client *HttpClient) SetTimeout(timeout time.Duration) {
	client.timeout = timeout
	client.httpCli.Timeout = timeout
}

func (client *HttpClient) GET(url string, data []byte) ([]byte, int, error) {
	return client.Request(url, http.MethodGet, data)
}

func (client *HttpClient) RawGET(url string, data []byte) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	// 设置请求头
	for key, value := range client.header {
		req.Header.Set(key, value)
	}
	resp, err := client.httpCli.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (client *HttpClient) POST(url string, data []byte) ([]byte, int, error) {
	return client.Request(url, http.MethodPost, data)
}

func (client *HttpClient) DELETE(url string, data []byte) ([]byte, int, error) {
	return client.Request(url, http.MethodDelete, data)
}

func (client *HttpClient) PUT(url string, data []byte) ([]byte, int, error) {
	return client.Request(url, http.MethodPut, data)
}

func (client *HttpClient) SetHeader(key, value string) {
	client.header[key] = value
}

func (client *HttpClient) SetQuery(key, value string) {
	client.query[key] = value
}

func (client *HttpClient) Request(url, method string, data []byte) ([]byte, int, error) {
	var req *http.Request
	var errReq error
	if data != nil {
		req, errReq = http.NewRequest(method, url, bytes.NewReader(data))
	} else {
		req, errReq = http.NewRequest(method, url, nil)
	}
	if errReq != nil {
		return nil, 0, errReq
	}
	req.Close = true
	for key, value := range client.header {
		req.Header.Set(key, value)
	}
	if method == http.MethodGet {
		q := req.URL.Query()
		for k, v := range client.query {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	if err := Sign(req, client.signCertificate); err != nil {
		return nil, 0, err
	}

	rsp, err := client.httpCli.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		if err = rsp.Body.Close(); err != nil {
			fmt.Printf("failed to close response body: %v", err)
		}
	}()
	body, err := io.ReadAll(rsp.Body)
	return body, rsp.StatusCode, err
}
