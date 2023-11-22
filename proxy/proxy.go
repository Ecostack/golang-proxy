package proxy

import (
	"bufio"
	"context"
	"errors"
	"go.uber.org/zap"
	"http-proxy/config"
	"http-proxy/otel_service"
	"http-proxy/utils"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

func SetupProxy(ctx context.Context) {
	listener, err := net.Listen("tcp", ":"+config.Port)
	if err != nil {
		otel_service.Fatal(ctx, otel_service.Logger, "[SetupProxy] Failed to set up listener:", zap.Error(err))
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			otel_service.Error(ctx, otel_service.Logger, "[SetupProxy] Error closing listener:", zap.Error(err))
		}
	}(listener)

	otel_service.Info(ctx, otel_service.Logger, "[SetupProxy] Proxy server listening on port "+config.Port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			otel_service.Error(ctx, otel_service.Logger, "[SetupProxy] Failed to accept connection:", zap.Error(err))
			continue
		}

		go handleConnection(conn)
	}
}

func getProxy() string {
	return strings.TrimSpace(utils.SelectWeighted(config.ParentProxy, config.ParentProxyWeight))
}

func handleConnection(clientConn net.Conn) {
	ctx := context.Background()
	ctx, span := otel_service.Tracer.Start(ctx, "handleConnection")
	defer span.End()

	defer func(clientConn net.Conn) {
		err := clientConn.Close()
		if err != nil {
			otel_service.Error(ctx, otel_service.Logger, "[handleConnection] Error closing clientConn:", zap.Error(err))
		}
	}(clientConn)

	reader := bufio.NewReader(clientConn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		otel_service.Error(ctx, otel_service.Logger, "[handleConnection] Failed to read request:", zap.Error(err))
		return
	}
	otel_service.Info(ctx, otel_service.Logger, "[handleConnection] connection", zap.String("req.Proto", req.Proto), zap.String("req.Method", req.Method), zap.String("req.URL", req.URL.String()))

	if req.Method == http.MethodConnect {
		handleHTTPS(ctx, clientConn, req)
	} else {
		handleHTTP(ctx, clientConn, req)
	}
}

func handleHTTPRequest(ctx context.Context, clientConn net.Conn, req *http.Request, retryCount uint) (*http.Response, error) {
	ctx, span := otel_service.Tracer.Start(ctx, "handleHTTPRequest")
	defer span.End()

	// Create a transport that uses the parent proxy
	proxy := getProxy()
	otel_service.Info(ctx, otel_service.Logger, "[handleHTTPRequest] Establishing tunnel with parent proxy", zap.String("proxy", proxy))
	transport := &http.Transport{
		Proxy: http.ProxyURL(&url.URL{
			Host: proxy,
		}),
	}

	// Forward the request to the destination
	resp, err := transport.RoundTrip(req)
	if err != nil {
		otel_service.Error(ctx, otel_service.Logger, "[handleHTTPRequest] Failed to forward request:", zap.Error(err))
		return nil, err
	}

	if resp.StatusCode == 407 {
		otel_service.Warn(ctx, otel_service.Logger, "[handleHTTPRequest] Received 407 from parent proxy")
		if config.RetryOnError {
			return handleHTTPRequest(ctx, clientConn, req, retryCount+1)
		}
	}
	return resp, nil
}

func handleHTTP(ctx context.Context, clientConn net.Conn, req *http.Request) {
	ctx, span := otel_service.Tracer.Start(ctx, "handleHTTP")
	defer span.End()

	resp, err := handleHTTPRequest(ctx, clientConn, req, 0)
	if err != nil {
		otel_service.Error(ctx, otel_service.Logger, "[handleHTTP] Failed to handle HTTP request:", zap.Error(err))
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			otel_service.Error(ctx, otel_service.Logger, "[handleHTTP] Error closing body:", zap.Error(err))
		}
	}(resp.Body)

	// Write the response back to the client
	err = resp.Write(clientConn)
	if err != nil {
		otel_service.Error(ctx, otel_service.Logger, "[handleHTTP] Failed to write response from parent proxy to client conn:", zap.Error(err))
		return
	}
}

func establishTunnel(ctx context.Context, clientConn net.Conn, req *http.Request, tryCounter uint) (net.Conn, error) {
	// Connect to the parent proxy
	ctx, span := otel_service.Tracer.Start(ctx, "establishTunnel")
	defer span.End()

	proxy := getProxy()
	otel_service.Info(ctx, otel_service.Logger, "[establishTunnel] Establishing tunnel with parent proxy", zap.String("proxy", proxy))
	proxyConn, err := net.Dial("tcp", proxy)
	if err != nil {
		otel_service.Error(ctx, otel_service.Logger, "[establishTunnel] Failed to connect to parent proxy:", zap.Error(err))
		return nil, err
	}

	closeProxyConn := func(ctx context.Context, proxyConn net.Conn) {
		errClose := proxyConn.Close()
		if errClose != nil {
			otel_service.Error(ctx, otel_service.Logger, "[establishTunnel] Error closing proxyConn:", zap.Error(errClose))
		}
	}

	// Send CONNECT request to the parent proxy
	connectReq := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Host: req.URL.Host},
		Host:   req.URL.Host,
		Header: make(http.Header),
	}

	err = connectReq.Write(proxyConn)
	if err != nil {
		otel_service.Error(ctx, otel_service.Logger, "[establishTunnel] Failed to write CONNECT request to parent proxy", zap.Error(err))
		closeProxyConn(ctx, proxyConn)
		return nil, err
	}

	// Read the response from the destination server or parent proxy
	resp, err := http.ReadResponse(bufio.NewReader(proxyConn), connectReq)
	if err != nil {
		otel_service.Error(ctx, otel_service.Logger, "[establishTunnel] Failed to read response", zap.Error(err))
		closeProxyConn(ctx, proxyConn)
		return nil, err
	}

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		err := errors.New("non-OK status from server: " + resp.Status)
		otel_service.Error(ctx, otel_service.Logger, "[establishTunnel] non-OK status from server ", zap.Error(err))

		if config.RetryOnError {
			if tryCounter < config.MaxRetryCount {
				tryCounter++
				closeProxyConn(ctx, proxyConn)
				otel_service.Warn(ctx, otel_service.Logger, "[establishTunnel] Retrying to establish tunnel ", zap.Uint("tryCounter", tryCounter))
				return establishTunnel(ctx, clientConn, req, tryCounter)
			} else {
				otel_service.Warn(ctx, otel_service.Logger, "[establishTunnel] Max retry count reached ")
			}
		}
		// Handle non-OK status appropriately
		closeProxyConn(ctx, proxyConn)
		return nil, err
	}
	return proxyConn, nil
}

func handleHTTPS(ctx context.Context, clientConn net.Conn, req *http.Request) {
	ctx, span := otel_service.Tracer.Start(ctx, "handleHTTPS")
	defer span.End()
	// Establish a tunnel with the parent proxy
	proxyConn, err := establishTunnel(ctx, clientConn, req, 0)
	if err != nil {
		otel_service.Error(ctx, otel_service.Logger, "[handleHTTPS] Failed to establish tunnel", zap.Error(err))
		return
	}

	defer func(proxyConn net.Conn) {
		err := proxyConn.Close()
		if err != nil {
			otel_service.Error(ctx, otel_service.Logger, "[handleHTTPS] Error closing proxyConn", zap.Error(err))
		}
	}(proxyConn)

	// If response is OK, start tunneling the traffic
	_, err = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		otel_service.Error(ctx, otel_service.Logger, "[handleHTTPS] Failed to write response to clientConn", zap.Error(err))
		return
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		//defer func(proxyConn net.Conn) {
		//	err := proxyConn.Close()
		//	if err != nil {
		//		log.Println("[handleHTTPS] Error closing proxyConn:", err)
		//	}
		//}(proxyConn)
		_, err = io.Copy(clientConn, proxyConn)
		if err != nil {
			otel_service.Error(ctx, otel_service.Logger, "[handleHTTPS] Failed to COPY proxyConn to clientConn request to parent proxy", zap.Error(err))
			//log.Printf("[handleHTTPS] Failed to COPY proxyConn to clientConn request to parent proxy: %v", err)
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		//defer func(clientConn net.Conn) {
		//	err := clientConn.Close()
		//	if err != nil {
		//		log.Println("[handleHTTPS] Error closing clientConn:", err)
		//	}
		//}(clientConn)
		_, err = io.Copy(proxyConn, clientConn)
		if err != nil {
			otel_service.Error(ctx, otel_service.Logger, "[handleHTTPS] Failed to COPY clientConn to proxyConn request to parent proxy", zap.Error(err))
			//log.Printf("[handleHTTPS] Failed to COPY clientConn to proxyConn request to parent proxy: %v", err)
			return
		}
	}()

	wg.Wait()
}
