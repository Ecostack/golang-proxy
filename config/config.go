package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
	"strings"
)

var Port = "3128"
var ParentProxy = make([]string, 0)
var ParentProxyWeight = make([]int, 0)
var RetryOnError = false

var MaxRetryCount uint = 5

func InitConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	port := os.Getenv("PORT")
	if port != "" {
		Port = port
	}

	parentProxy := os.Getenv("PARENT_PROXY")
	if parentProxy != "" {
		proxyDefinitions := strings.Split(parentProxy, ",")
		for _, proxyDefSingle := range proxyDefinitions {
			proxyDefSplit := strings.Split(proxyDefSingle, "=")
			if len(proxyDefSplit) == 2 {
				ParentProxy = append(ParentProxy, proxyDefSplit[0])
				weight, err := strconv.Atoi(proxyDefSplit[1])
				if err != nil {
					log.Fatal("Error parsing PARENT_PROXY_WEIGHT", err)
				}
				ParentProxyWeight = append(ParentProxyWeight, weight)
			} else if len(proxyDefSplit) == 1 {
				ParentProxy = append(ParentProxy, proxyDefSplit[0])
				ParentProxyWeight = append(ParentProxyWeight, 1)
			} else {
				log.Fatal("Error parsing PARENT_PROXY: ", proxyDefSingle)
			}
		}
	}

	if len(ParentProxy) == 0 {
		log.Fatal("PARENT_PROXY not set")
	}

	retryOnError := os.Getenv("RETRY_ON_ERROR")
	if strings.ToUpper(retryOnError) == "TRUE" {
		RetryOnError = true
	}

	log.Println("Config loaded")
	log.Println("Port:", Port)
	log.Println("ParentProxy: ", ParentProxy)
	log.Println("ParentProxyWeight: ", ParentProxyWeight)
	log.Println("RetryOnError: ", RetryOnError)
	log.Println("MaxRetryCount: ", MaxRetryCount)
}
