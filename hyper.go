package hyper

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Clienter interface {
	Do(*http.Request) (*http.Response, error)
}

var defaultClient = http.DefaultClient

func Check200(r *http.Response) error {
	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("[%v] %v", r.StatusCode, r.Status)
	}
	return nil
}

type Request struct {
	client          Clienter
	request         *http.Request
	err             error
	onResponseCheck func(*http.Response) error
}

type Response struct {
	*http.Response
}

func (r *Request) OnResponseCheck(f func(*http.Response) error) *Request {
	r.onResponseCheck = f
	return r
}

func (r *Response) Raw() ([]byte, error) {
	defer r.Body.Close()
	return io.ReadAll(r.Body)
}

func (r *Response) ParseJson(i any) error {
	err := json.NewDecoder(r.Body).Decode(i)
	if err != nil {
		return err
	}
	return r.Body.Close()
}

func New() *Request {
	return &Request{
		request: &http.Request{
			Header: make(http.Header),
		},
	}
}

func (r *Request) Get() *Request {
	r.request.Method = http.MethodGet
	return r
}

func (r *Request) Post() *Request {
	r.request.Method = http.MethodPost
	return r
}

func (r *Request) Delete() *Request {
	r.request.Method = http.MethodDelete
	return r
}

func (r *Request) Put() *Request {
	r.request.Method = http.MethodPut
	return r
}

func (r *Request) Patch() *Request {
	r.request.Method = http.MethodPatch
	return r
}

func (r *Request) Options() *Request {
	r.request.Method = http.MethodOptions
	return r
}

func (r *Request) SetClient(client Clienter) *Request {
	r.client = client
	return r
}

func (r *Request) SetHeader(key string, values ...string) *Request {
	if len(values) == 0 {
		r.err = fmt.Errorf("missing values for set header %q", key)
		return r
	}
	for index, value := range values {
		if index == 0 {
			r.request.Header.Set(key, value)
		} else {
			r.request.Header.Add(key, value)

		}
	}
	return r
}

func (r *Request) SetQueryParam(key, value string) *Request {
	if r.request.URL == nil {
		r.err = errors.New("cannot add query param to nil url")
		return r
	}
	query := r.request.URL.Query()
	query.Add(key, value)
	r.request.URL.RawQuery = query.Encode()
	return r
}

func (r *Request) GetHeader() http.Header {
	return r.request.Header
}

func (r *Request) Url(u string) *Request {
	res, err := url.Parse(u)
	if err != nil {
		r.err = err
		return r
	}
	r.request.URL = res
	return r
}

func (r *Request) Body(rc io.ReadCloser) *Request {
	r.request.Body = rc
	return r
}

func (r *Request) Json(body any) *Request {
	r.SetHeader("content-type", "application/json")

	data, err := json.Marshal(body)
	if err != nil {
		r.err = err
		return r
	}
	buf := bytes.NewBuffer(data)
	bufclose := io.NopCloser(buf)
	r.request.Body = bufclose
	return r
}

func (r *Request) DoAndParseJson(i any) error {
	resp, err := r.Do()
	if err != nil {
		if resp != nil {
			raw, err2 := io.ReadAll(resp.Body)
			if err2 != nil {
				return fmt.Errorf("error: %v, content=%v", err, err2)
			} else {
				return fmt.Errorf("error: %v, content=%v", err, string(raw))
			}
		}
		return fmt.Errorf("error: %v", err)
	}
	return resp.ParseJson(i)
}

func (r *Request) Do() (*Response, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.client == nil {
		r.client = defaultClient
	}
	resp, err := r.client.Do(r.request)
	if err != nil {
		return nil, err
	}
	if r.onResponseCheck != nil {
		err = r.onResponseCheck(resp)
		if err != nil {
			return nil, err
		}
	}
	return &Response{resp}, nil
}

func (r *Request) Context(ctx context.Context) *Request {
	r.request = r.request.WithContext(ctx)
	return r
}

func (r *Request) Clone() *Request {
	return r.CloneWithContext(r.request.Context())
}

func (r *Request) CloneWithContext(ctx context.Context) *Request {
	return &Request{
		request: r.request.Clone(ctx),
		err:     r.err,
		client:  r.client,
	}
}
