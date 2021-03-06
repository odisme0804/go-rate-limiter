package main

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"go-rate-limiter/internal/cache"
	"go-rate-limiter/internal/limiter"
)

var rateLimiter limiter.Limiter

type requestHandlerRsp struct {
	CurrentCnt int   `json:"current_cnt"`
	Expiration int64 `json:"expiration"`
}

type requestHandlerErrRsp struct {
	Error string `json:"error"`
}

func init() {
	cache := cache.NewInMemoryCache(time.Minute)
	rateLimiter = limiter.NewTokenBucketLimiter(time.Minute, 60, cache)
}

func main() {
	mux := http.NewServeMux()

	mux.Handle("/request", requestHandler)

	log.Println("Listening...")
	if err := http.ListenAndServe("0.0.0.0:8080", mux); err != nil {
		panic(err)
	}
}

var requestHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ipAddr := getRequestIP(r)
	currentCnt, expiration, err := rateLimiter.Take(ipAddr)
	log.Printf("ip: %s, req #: %d, exp: %d", ipAddr, currentCnt, expiration)
	if err != nil {
		if errors.Is(limiter.ErrReachLimit, err) {
			rsp := requestHandlerErrRsp{
				Error: "reach request limit",
			}
			bs, err := json.Marshal(rsp)
			if err != nil {
				log.Printf("[requestHandler] err: %+v, rsp: %+v", err, rsp)
			}
			w.Header().Set("Retry-After", strconv.FormatInt(expiration-time.Now().Unix(), 10))
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write(bs)
			return
		}
	}
	rsp := requestHandlerRsp{
		CurrentCnt: currentCnt,
		Expiration: expiration,
	}
	bs, err := json.Marshal(rsp)
	if err != nil {
		log.Printf("[requestHandler] json.Marshal err: %+v, rsp: %+v", err, rsp)
	}
	w.Write(bs)
}

func getRequestIP(r *http.Request) string {
	if xRealIP := r.Header.Get("X-REAL-IP"); xRealIP != "" {
		return net.ParseIP(xRealIP).String()
	} else if xff := r.Header.Get("X-FORWARDED-FOR"); xff != "" {
		return net.ParseIP(xff).String()
	}

	return dropPort(r.RemoteAddr)
}

func dropPort(addr string) string {
	ip, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}

	return ip
}
