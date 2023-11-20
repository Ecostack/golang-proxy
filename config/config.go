package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

var Port = "3128"
var ParentProxy = ""

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
		ParentProxy = parentProxy
	}

	if ParentProxy == "" {
		log.Fatal("PARENT_PROXY not set")
	}
}
