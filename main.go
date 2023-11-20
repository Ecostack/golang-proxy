package main

import (
	"http-proxy/config"
	"http-proxy/parent_proxy"
)

func main() {
	config.InitConfig()
	parent_proxy.SetupProxy()
}
