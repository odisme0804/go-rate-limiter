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

type pingHandlerRsp struct {
	CurrentCnt int   `json:"current_cnt"`
	Expiration int64 `json:"expiration"`
}

type pingHandlerErrRsp struct {
	Error string `json:"error"`
}

func init() {
	cache := cache.NewInMemoryCache(time.Minute)
	rateLimiter = limiter.NewTokenBucketLimiter(time.Minute, 60, cache)
}

func main() {
	mux := http.NewServeMux()

	mux.Handle("/", homeHandler)
	mux.Handle("/ping", pingHandler)

	log.Println("Listening...")
	if err := http.ListenAndServe("0.0.0.0:8080", mux); err != nil {
		panic(err)
	}
}

var homeHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
	ipAddr := getRequestIP(r)
	currentCnt, expiration, err := rateLimiter.Check(ipAddr)
	rsp := pingHandlerRsp{
		CurrentCnt: currentCnt,
		Expiration: expiration,
	}
	bs, err := json.Marshal(rsp)
	if err != nil {
		log.Printf("[homeHandler] json.Marshal err: %+v, rsp: %+v", err, rsp)
	}
	w.Write(bs)
}

var pingHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
	ipAddr := getRequestIP(r)
	log.Printf("ping from: %s", ipAddr)
	currentCnt, expiration, err := rateLimiter.Take(ipAddr)
	if err != nil {
		if errors.Is(limiter.ErrReachLimit, err) {
			rsp := pingHandlerErrRsp{
				Error: "reach request limit",
			}
			bs, err := json.Marshal(rsp)
			if err != nil {
				log.Printf("[pingHandler] err: %+v, rsp: %+v", err, rsp)
			}
			w.Header().Set("Retry-After", strconv.FormatInt(expiration-time.Now().Unix(), 10))
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write(bs)
			return
		}
	}
	rsp := pingHandlerRsp{
		CurrentCnt: currentCnt,
		Expiration: expiration,
	}
	bs, err := json.Marshal(rsp)
	if err != nil {
		log.Printf("[pingHandler] json.Marshal err: %+v, rsp: %+v", err, rsp)
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
