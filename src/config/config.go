package config

import (
	"log"
	"os"
	"reflect"

	"github.com/joho/godotenv"
)

type Config struct {
	API_TOKEN   string `env:"API_TOKEN"`
	SD_TOKEN    string `env:"SD_TOKEN"`
	LISTEN_PORT string `env:"LISTEN_PORT"`
	REDIS_ADDR  string `env:"REDIS_ADDR"`
	REDIS_PORT  string `env:"REDIS_PORT"`
	REDIS_DBNO  string `env:"REDIS_DBNO"`
	TTL_SECONDS string `env:"TTL_SECONDS"`
}

var config *Config

func GetConfig() Config {
	return *config
}

func loadEnvFile() {
	log.Println("Loading .env file.")
	if err := godotenv.Load(".env"); err != nil {
		panic("Error loading .env file.")
	}
}

func init() {
	if _, found := os.LookupEnv("APP_ENV"); !found {
		loadEnvFile()
	}
	if _, found := os.LookupEnv("LISTEN_PORT"); !found {
		os.Setenv("LISTEN_PORT", "9101")
	}
	if _, found := os.LookupEnv("SD_TOKEN"); !found {
		os.Setenv("SD_TOKEN", "")
	}

	config = &Config{}
	refl := reflect.ValueOf(config).Elem()
	for i := 0; i < refl.NumField(); i++ {
		envName := refl.Type().Field(i).Tag.Get("env")
		envVal, found := os.LookupEnv(envName)
		if !found {
			panic("Environment [" + envName + "] not found.")
		}
		refl.Field(i).SetString(envVal)
	}
}
