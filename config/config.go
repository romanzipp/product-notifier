package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type Config struct {
	LoopInterval int       `json:"loop_interval"`
	Products     []Product `json:"products"`
}

type Product struct {
	Title     string     `json:"title"`
	Image     string     `json:"img"`
	Sizes     []string   `json:"sizes"`
	Providers []Provider `json:"providers"`
}

type Provider struct {
	Id  string `json:"id"`
	Url string `json:"url"`
}

func ReadConfig() *Config {
	content, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatal(err)
	}

	cfg := &Config{}

	if err := json.Unmarshal(content, cfg); err != nil {
		log.Fatal(err)
	}

	return cfg
}
