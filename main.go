package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

type Route struct {
	Pattern string `yaml:"pattern"`
	Pass    string `yaml:"pass"`
}

type Virtualhost struct {
	Host    string   `yaml:"host"`
	Aliases []string `yaml:"aliases"`
	Routes  []Route  `yaml:"routes"`
}

type Config struct {
	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`
	Virtualhosts []Virtualhost `yaml:"virtualhosts"`
}

func (c *Config) handler(w http.ResponseWriter, req *http.Request) {
	hostname, _, err := net.SplitHostPort(req.Host)
	if err != nil {
		fmt.Println("Could not split port")
	}

	fmt.Println(hostname)
	var matchedHost *Virtualhost
	for i := range c.Virtualhosts {
		vh := &c.Virtualhosts[i]
		fmt.Println(vh.Host)
		if vh.Host == hostname {
			fmt.Println("match on", vh.Host)
			matchedHost = vh
			break
		}

		for _, alias := range vh.Aliases {
			if alias == hostname {
				fmt.Println("match on", alias)
				matchedHost = vh
				break
			}
		}
	}

	if matchedHost == nil {
		return
	}

	var matchedRoute *Route
	for i := range matchedHost.Routes {
		route := &matchedHost.Routes[i]
		r, err := regexp.Compile(route.Pattern)
		if err != nil {
			fmt.Println("Unable to compile regex for", route.Pattern)
			continue
		}
		if r.MatchString(req.URL.String()) {
			matchedRoute = route
		}

	}

	fmt.Println(matchedRoute.Pass)
}

func main() {
	yamlFile, err := os.ReadFile("config.yml")
	if err != nil {
		log.Printf("Error reading YAML file: %v", err)
		return
	}

	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Printf("Error unmarshaling YAML: %v", err)
		return
	}
	http.HandleFunc("/", config.handler)

	fmt.Println(config.Virtualhosts)
	for _, virtualhost := range config.Virtualhosts {
		fmt.Println(virtualhost.Host)
	}

	fmt.Println("Server started on port", config.Server.Port)
	http.ListenAndServe(fmt.Sprintf(":%d", config.Server.Port), nil)
}
