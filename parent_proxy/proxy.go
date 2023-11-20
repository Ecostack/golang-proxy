package parent_proxy

import (
	"bufio"
	"http-proxy/config"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
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

func handleHTTP(clientConn net.Conn, req *http.Request) {

	// Create a transport that uses the parent proxy
	transport := &http.Transport{
		Proxy: http.ProxyURL(&url.URL{
			Host: config.ParentProxy,
		}),
	}

	// Forward the request to the destination
	resp, err := transport.RoundTrip(req)
	if err != nil {
		log.Printf("[handleHTTP] Failed to forward request: %v", err)
		return
	}
	defer resp.Body.Close()

	// Reading the status code
	//statusCode := resp.StatusCode
	//log.Printf("Received status code: %d", statusCode)
	if resp.StatusCode != http.StatusOK {
		log.Printf("[handleHTTP] Received non-OK status from server: %s", resp.Status)
		// Handle non-OK status appropriately
		return
	}

	// Write the response back to the client
	err = resp.Write(clientConn)
	if err != nil {
		log.Printf("[handleHTTP] Failed to write response from parent proxy to client conn: %v", err)
		return
	}
}

func handleHTTPS(clientConn net.Conn, req *http.Request) {
	// Connect to the parent proxy
	proxyConn, err := net.Dial("tcp", config.ParentProxy)
	if err != nil {
		log.Printf("Failed to connect to parent proxy: %v", err)
		return
	}

	defer proxyConn.Close()

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
		return
	}

	// Read the response from the destination server or parent proxy
	resp, err := http.ReadResponse(bufio.NewReader(proxyConn), connectReq)
	if err != nil {
		log.Printf("[handleHTTPS] Failed to read response: %v", err)
		return
	}

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		log.Printf("[handleHTTPS] Received non-OK status from server: %s", resp.Status)
		// Handle non-OK status appropriately
		return
	}

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
