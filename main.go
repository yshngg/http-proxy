package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"

	"k8s.io/klog/v2"
)

func main() {
	addr, cert, key, ca := ":1080", "", "", ""
	flag.StringVar(&addr, "addr", addr, "Address the proxy server will listen on.")
	flag.StringVar(&cert, "cert", cert, "(TLS) Use the specified certificate file when run start proxy server with HTTPS")
	flag.StringVar(&key, "key", key, "(TLS) Use the specified key file when run start proxy server with HTTPS")
	flag.StringVar(&ca, "ca", ca, "")
	flag.Parse()

	if err := run(addr, cert, key, ca); err != nil {
		klog.Fatal(err)
	}
}

func run(addr, certFile, keyFile, caFile string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s, err: %w", addr, err)
	}

	if len(certFile) != 0 && len(keyFile) != 0 {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return fmt.Errorf("load x509 key pair, err: %w", err)
		}

		cfg := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		if len(caFile) != 0 {
			caPem, err := os.ReadFile(caFile)
			if err != nil {
				return fmt.Errorf("read CA file, err: %w", err)
			}
			block, _ := pem.Decode(caPem)
			caCert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return fmt.Errorf("parse x509 CA cert, err: %w", err)
			}
			pool, err := x509.SystemCertPool()
			if err != nil {
				return fmt.Errorf("get system cert poll, err: %w", err)
			}
			pool.AddCert(caCert)
			cfg.RootCAs = pool

			klog.Infof("use CA cert file: %s", caFile)
		}

		l = tls.NewListener(l, cfg)
		klog.Infof("use cert file: %s and key file: %s", certFile, keyFile)
	}

	defer l.Close()
	klog.Infoln("listen on:", l.Addr())

	for {
		localConn, err := l.Accept()
		if err != nil {
			klog.Warningf("accept connection, err: %v", err)
			continue
		}
		go func() {
			err := handleConn(localConn)
			if err != nil {
				klog.Warningf("handle connection, err: %v", err)
			}
		}()
	}
}

func handleConn(conn net.Conn) error {
	defer conn.Close()
	localReader := bufio.NewReader(conn)
	req, err := http.ReadRequest(localReader)
	if err != nil {
		return err
	}
	klog.Infoln("Request:", req.Method, req.Proto, req.Host)
	for k, v := range req.Header {
		klog.Infof("< %s: %v", k, v)
	}

	host := req.Host
	if req.URL.Port() == "" && req.TLS == nil {
		host += ":80"
	}

	remoteConn, err := net.Dial("tcp", host)
	if err != nil {
		return err
	}
	defer remoteConn.Close()

	if req.Method == http.MethodConnect {
		resp := http.Response{
			Proto:      req.Proto,
			ProtoMajor: req.ProtoMajor,
			ProtoMinor: req.ProtoMinor,
			StatusCode: http.StatusOK,
		}
		resp.Write(conn)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			io.Copy(remoteConn, conn)
		}()
		go func() {
			defer wg.Done()
			io.Copy(conn, remoteConn)
		}()
		wg.Wait()
		return nil
	}

	remoteReader := bufio.NewReader(remoteConn)
	err = req.Write(remoteConn)
	if err != nil {
		return err
	}
	resp, err := http.ReadResponse(remoteReader, req)
	if err != nil {
		return err
	}
	klog.Infoln("Response:", resp.StatusCode, resp.Proto)
	for k, v := range resp.Header {
		klog.Infof("> %s: %v", k, v)
	}
	resp.Write(conn)
	return nil
}
