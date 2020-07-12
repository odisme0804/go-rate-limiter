# go-rate-limiter
A simple rate limit server (60 req/minute) through token bucket algorithm to control requests' limitation of each IP address.


## Prerequisite

- Go v1.14

## Usage
You can run the server code by:

```bash
make dev
```
#
Or, you can also build it and run by:
```bash
make run
```
the server will run on `localhost:8080`

## Test

Generate a request: 
```bash
$ curl -i -X GET localhost:8080/request
```
or use a browser and goto:

```bash
localhost:8080/request
```

You'll get the response like:
```javascript
HTTP/1.1 200 OK
Date: Sun, 12 Jul 2020 14:30:21 GMT
Content-Length: 42
Content-Type: text/plain; charset=utf-8

{"current_cnt":1,"expiration":1594564269}
```
`current_cnt`: how many requests which you have sent after first request within the time window (60 seconds).  
`expiration`: the unix timestamp of refresh time.  

#
If you request more than 60 times within one minute, you'll get:
```javascript
HTTP/1.1 429 Too Many Requests
Retry-After: 53
Date: Sun, 12 Jul 2020 14:33:36 GMT
Content-Length: 31
Content-Type: text/plain; charset=utf-8

{"error":"reach request limit"}
```
#
You can send multiple requests in parallel by:
```bash
$ ./script/multi_request.sh {number-of-request}
```
for example:
```bash
$ ./script/multi_request.sh 60
```
you'll reach the limit immediately.