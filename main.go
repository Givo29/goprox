package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"

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

	var matchedHost *Virtualhost
	for i := range c.Virtualhosts {
		vh := &c.Virtualhosts[i]
		if vh.Host == hostname {
			matchedHost = vh
			break
		}

		if slices.Contains(vh.Aliases, hostname) {
			matchedHost = vh
			break
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

	if matchedRoute == nil {
		return
	}

	resp := proxyRequest(*req, *matchedRoute)
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Content-Encoding", resp.Header.Get("Content-Encoding"))
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Fprint(w, string(bodyBytes))
}

func proxyRequest(req http.Request, matchedRoute Route) http.Response {
	newUrl, err := url.Parse(matchedRoute.Pass)
	if err != nil {
		log.Fatalln("Unable to parse URL of matched route:", matchedRoute.Pass, err)
	}
	req.RequestURI = ""
	req.Host = newUrl.Host
	req.URL = newUrl

	client := &http.Client{}

	resp, err := client.Do(&req)
	if err != nil {
		log.Fatalln("Unable to send request:", err)
	}

	return *resp
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

	fmt.Println("Server started on port", config.Server.Port)
	http.ListenAndServe(fmt.Sprintf(":%d", config.Server.Port), nil)
}
