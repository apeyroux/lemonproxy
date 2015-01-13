package main

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

var (
	flcf *string = flag.String("c", "", "Config file")
)

type Configuration struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	AuthSSO  string `json:"authsso"`
	ProxyTo  string `json:"proxyto"`
	Listen   string `json:"listen"`
}

func newConfig(path string) (config Configuration, err error) {
	cf, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	err = json.Unmarshal(cf, &config)
	return
}

func main() {

	flag.Parse()

	if *flcf == "" {
		log.Fatal("Voir le -h")
	}

	cf, err := newConfig(*flcf)

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("cnx sso with %s", cf.Login)

	urlProxyTo, err := url.Parse(cf.ProxyTo)
	if err != nil {
		log.Fatal(err)
	}

	urlAuthSSO, err := url.Parse(cf.AuthSSO)
	if err != nil {
		log.Fatal(err)
	}

	tr := &http.Transport{
		DisableCompression: true,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{Transport: tr}

	userInfo := url.Values{}
	userInfo.Add("url", base64.StdEncoding.EncodeToString([]byte(urlProxyTo.String())))
	userInfo.Add("timezone", "1")
	userInfo.Add("user", cf.Login)
	userInfo.Add("password", cf.Password)

	resp, err := client.PostForm(urlAuthSSO.String(), userInfo)
	if err != nil {
		log.Fatalf("err:%s", err)
	}
	defer resp.Body.Close()

	sourceAddress := cf.Listen
	director := func(req *http.Request) {
		req.URL.Scheme = urlProxyTo.Scheme
		req.URL.Host = urlProxyTo.Host
		req.Host = urlProxyTo.Host

		log.Printf("load cookies ...")
		for _, c := range resp.Cookies() {
			log.Printf("%s:%s", c.Name, c.Value)
			req.AddCookie(c)
		}
	}

	proxy := &httputil.ReverseProxy{Director: director}
	server := http.Server{
		Addr:    sourceAddress,
		Handler: proxy,
	}

	log.Printf("listen %s", cf.Listen)
	err = server.ListenAndServe()

	if err != nil {
		log.Fatal(err)
	}
}