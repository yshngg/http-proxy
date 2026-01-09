# httpproxy-go

A HTTP proxy server with CONNECT request method.

- [x] HTTP/HTTPS 1.1
- [ ] HTTP2
- [ ] HTTP3

```bash
# Run proxy server:
go run . --addr :8088

# Verify:
curl -x http://localhost:8088 -vI https://example.org
```

```bash
# Run proxy server:
go run . --addr :8088 --cert ./pki/server.pem --key ./pki/server-key.pem  --ca ./pki/ca.pem

# Verify:
curl -x https://localhost:8088 --proxy-cacert ./pki/server.pem -vI https://example.org
# or
curl --proxy-insecure -x https://localhost:8088 --proxy-cacert ./pki/ca.pem -vI https://example.org
```
