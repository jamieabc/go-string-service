package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/ratelimit"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/lb"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
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

func proxyingMiddleware(ctx context.Context, instances string, logger log.Logger) ServiceMiddleware {
	if "" == instances {
		logger.Log("proxy_to", "non")
		return func(next StringService) StringService { return next }
	}

	var (
		qps         = 100
		maxAttempts = 3
		maxTime     = 250 * time.Millisecond
	)

	var (
		instanceList = split(instances)
		subscriber   sd.FixedEndpointer
	)

	logger.Log("proxy_to", fmt.Sprint(instanceList))
	for _, instance := range instanceList {
		var e endpoint.Endpoint
		e = makeUppercaseProxy(ctx, instance)
		e = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(e)
		e = ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Every(time.Second), qps))(e)
		subscriber = append(subscriber, e)
	}

	balancer := lb.NewRoundRobin(subscriber)
	retry := lb.Retry(maxAttempts, maxTime, balancer)

	return func(next StringService) StringService {
		return proxymw{ctx, next, retry}
	}
}

func makeUppercaseProxy(ctx context.Context, instance string) endpoint.Endpoint {
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}

	url, err := url.Parse(instance)
	if nil != err {
		panic(err)
	}

	if "" == url.Path {
		url.Path = "/uppercase"
	}

	return httptransport.NewClient(
		"GET",
		url,
		encodeUppercaseRequest,
		decodeUppercaseResponse,
	).Endpoint()
}

func split(s string) []string {
	a := strings.Split(s, ",")
	for i := range a {
		a[i] = strings.TrimSpace(a[i])
	}
	return a
}
