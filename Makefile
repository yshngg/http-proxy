CFSSL = go tool github.com/cloudflare/cfssl/cmd/cfssl
CFSSLJSON = go tool github.com/cloudflare/cfssl/cmd/cfssljson

.PHONY: ssl
ssl:
	# generate certificates manually
	@mkdir -p ./pki
	@$(CFSSL) gencert -initca ./ca-csr.json | $(CFSSLJSON) -bare ./pki/ca
	@$(CFSSL) gencert -ca=./pki/ca.pem -ca-key=./pki/ca-key.pem \
	--config=./ca-config.json -profile=http-proxy \
	./server-csr.json | $(CFSSLJSON) -bare ./pki/server

	# distributing self-signed CA certificate
	@sudo mkdir -p /usr/local/share/ca-certificates
	@sudo cp ./pki/ca.pem /usr/local/share/ca-certificates/proxy-server.crt
	@sudo update-ca-certificates

clean:
	@rm -r ./pki
	@sudo rm /usr/local/share/ca-certificates/proxy-server.crt
	@sudo update-ca-certificates
