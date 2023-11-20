package main

import (
	"http-proxy/config"
	"http-proxy/proxy"
)

func main() {
	config.InitConfig()
	proxy.SetupProxy()
}
