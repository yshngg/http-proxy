package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"sync"
)

func main() {
	l, err := net.Listen("tcp", ":1080")
	if err != nil {
		log.Fatal(err)
	}

	for {
		localConn, err := l.Accept()
		if err != nil {
			slog.Warn("accept connection", slog.Attr{Key: "err", Value: slog.AnyValue(err)})
			continue
		}
		go func() {
			err := handleConn(localConn)
			if err != nil {
				slog.Warn("handle connection", slog.Attr{Key: "err", Value: slog.AnyValue(err)})
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
	fmt.Println("Request:", req.Method, req.Proto, req.Host)
	for k, v := range req.Header {
		fmt.Println("<", k, v)
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
	fmt.Println("Response:", resp.StatusCode, resp.Proto)
	for k, v := range resp.Header {
		fmt.Println(">", k, v)
	}
	resp.Write(conn)
	return nil
}
