package proxy

import (
	"bufio"
	"http-proxy/config"
	"http-proxy/util"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

func SetupProxy() {
	listener, err := net.Listen("tcp", ":"+config.Port)
	if err != nil {
		log.Fatalf("Failed to set up listener: %v", err)
	}
	defer listener.Close()

	log.Println("Proxy server listening on port " + config.Port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		go handleConnection(conn)
	}
}

func getProxy() string {
	return strings.TrimSpace(util.PickRandomString(config.ParentProxy))
}

func handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	reader := bufio.NewReader(clientConn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Printf("Failed to read request: %v", err)
		return
	}
	log.Println(req.Proto, req.Method, req.URL)

	if req.Method == http.MethodConnect {
		handleHTTPS(clientConn, req)
	} else {
		handleHTTP(clientConn, req)
	}
}

func handleHTTPRequest(clientConn net.Conn, req *http.Request, retryCount uint) (*http.Response, error) {
	// Create a transport that uses the parent proxy
	proxy := getProxy()
	log.Println("[handleHTTPRequest] Using proxy: " + proxy)
	transport := &http.Transport{
		Proxy: http.ProxyURL(&url.URL{
			Host: proxy,
		}),
	}

	// Forward the request to the destination
	resp, err := transport.RoundTrip(req)
	if err != nil {
		log.Printf("[handleHTTPRequest] Failed to forward request: %v", err)
		return nil, err
	}

	if resp.StatusCode == 407 {
		log.Println("[handleHTTPRequest] Received 407 from parent proxy")
		if config.RetryOnError {
			return handleHTTPRequest(clientConn, req, retryCount+1)
		}
	}
	return resp, nil
}

func handleHTTP(clientConn net.Conn, req *http.Request) {

	resp, err := handleHTTPRequest(clientConn, req, 0)
	if err != nil {
		log.Printf("[handleHTTP] Failed to handle HTTP request: %v", err)
		return
	}
	defer resp.Body.Close()

	// Write the response back to the client
	err = resp.Write(clientConn)
	if err != nil {
		log.Printf("[handleHTTP] Failed to write response from parent proxy to client conn: %v", err)
		return
	}
}

func establishTunnel(clientConn net.Conn, req *http.Request, tryCounter uint) (net.Conn, error) {
	// Connect to the parent proxy

	proxy := getProxy()
	log.Println("[establishTunnel] Using proxy: " + proxy)
	proxyConn, err := net.Dial("tcp", proxy)
	if err != nil {
		log.Printf("Failed to connect to parent proxy: %v", err)
		return nil, err
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
		log.Printf("[handleHTTPS] Failed to write CONNECT request to parent proxy: %v", err)
		proxyConn.Close()
		return nil, err
	}

	// Read the response from the destination server or parent proxy
	resp, err := http.ReadResponse(bufio.NewReader(proxyConn), connectReq)
	if err != nil {
		log.Printf("[handleHTTPS] Failed to read response: %v", err)
		proxyConn.Close()
		return nil, err
	}

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		log.Printf("[handleHTTPS] Received non-OK status from server: %s", resp.Status)

		if config.RetryOnError {
			if tryCounter < config.MaxRetryCount {
				tryCounter++
				proxyConn.Close()
				log.Println("[handleHTTPS] Retrying to establish tunnel " + strconv.Itoa(int(tryCounter)))
				return establishTunnel(clientConn, req, tryCounter)
			}
		}
		// Handle non-OK status appropriately
		proxyConn.Close()
		return nil, err
	}

	return proxyConn, nil
}

func handleHTTPS(clientConn net.Conn, req *http.Request) {

	// Establish a tunnel with the parent proxy
	proxyConn, err := establishTunnel(clientConn, req, 0)
	if err != nil {
		log.Printf("[handleHTTPS] Failed to establish tunnel: %v", err)
		return
	}
	defer proxyConn.Close()

	// If response is OK, start tunneling the traffic
	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer proxyConn.Close()
		_, err = io.Copy(clientConn, proxyConn)
		if err != nil {
			log.Printf("[handleHTTPS] Failed to COPY proxyConn to clientConn request to parent proxy: %v", err)
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer clientConn.Close()
		_, err = io.Copy(proxyConn, clientConn)
		if err != nil {
			log.Printf("[handleHTTPS] Failed to COPY clientConn to proxyConn request to parent proxy: %v", err)
			return
		}
	}()

	wg.Wait()
}

//
//
//func handleHTTPS(clientConn net.Conn, req *http.Request) {
//	log.Println("Handling HTTPS request for:", req.URL.Host)
//
//	// Connect to the parent proxy
//	proxyConn, err := net.Dial("tcp", parentProxy)
//	if err != nil {
//		log.Printf("Failed to connect to parent proxy: %v", err)
//		return
//	}
//	log.Println("CONNECT to parent proxy successful")
//
//	defer func(proxyConn net.Conn) {
//		err := proxyConn.Close()
//		if err != nil {
//			log.Printf("Failed to close connection to parent proxy: %v", err)
//		}
//	}(proxyConn)
//
//	// Send CONNECT request to the parent proxy
//	connectReq := &http.Request{
//		Method: http.MethodConnect,
//		URL:    &url.URL{Host: req.URL.Host},
//		Host:   req.URL.Host,
//		Header: make(http.Header),
//	}
//	err = connectReq.Write(proxyConn)
//	if err != nil {
//		log.Printf("Failed to write CONNECT request to parent proxy: %v", err)
//	}
//
//	// Read the response from the parent proxy
//	br := bufio.NewReader(proxyConn)
//	resp, err := http.ReadResponse(br, connectReq)
//	if err != nil {
//		log.Printf("Failed to read response from parent proxy: %v", err)
//		return
//	}
//	if resp.StatusCode != http.StatusOK {
//		log.Printf("Non-OK response from parent proxy: %v", resp.Status)
//		return
//	}
//
//	log.Printf("Received response from parent proxy: %v", resp.Status)
//
//	// Tunnel the traffic
//	go func() {
//		_, err := io.Copy(proxyConn, clientConn)
//		if err != nil {
//			log.Printf("Error copying from client to parent proxy: %v", err)
//		}
//	}()
//	_, err = io.Copy(clientConn, proxyConn)
//	if err != nil {
//		log.Printf("Error copying from parent proxy to client: %v", err)
//		return
//	}
//}
