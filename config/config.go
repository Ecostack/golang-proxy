package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"strings"
)

var Port = "3128"
var ParentProxy = make([]string, 0)
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
		ParentProxy = strings.Split(parentProxy, ",")
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
	log.Println("RetryOnError: ", RetryOnError)
	log.Println("MaxRetryCount: ", MaxRetryCount)
}
