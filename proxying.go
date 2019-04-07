package main

import (
	"context"
	"errors"
	"net/url"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
)

type proxymw struct {
	ctx       context.Context
	next      StringService
	uppercase endpoint.Endpoint
}

func (mw proxymw) Uppercase(s string) (string, error) {
	response, err := mw.uppercase(mw.ctx, uppercaseRequest{S: s})
	if nil != err {
		return "", err
	}
	resp := response.(uppercaseResponse)
	if "" != resp.Err {
		return resp.V, errors.New(resp.Err)
	}
	return resp.V, nil
}

func (mw proxymw) Count(s string) int {
	return mw.next.Count(s)
}

func proxyingMiddleware(ctx context.Context, proxyURL string) ServiceMiddleware {
	return func(next StringService) StringService {
		return proxymw{ctx, next, makeUppercaseProxy(proxyURL)}
	}
}

func makeUppercaseProxy(proxyURL string) endpoint.Endpoint {
	url, _ := url.Parse(proxyURL)
	return httptransport.NewClient(
		"GET",
		url,
		encodeUppercaseRequest,
		decodeUppercaseResponse,
	).Endpoint()
}
