package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/gob"
	"errors"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var urlsPath = flag.String("urls", "urls.txt", "urls file path")
var nDownloaders = flag.Int("nd", 8, "number of downloaders")

var cacheFile = flag.String("cache", "resolutions.gob", "cache file")
var cacheHosts map[string]string

var statusLogger = log.New(os.Stdout, "", 0)
var errorLogger = log.New(os.Stderr, "", 0)

func readCache() {
	fin, err := os.Open(*cacheFile)
	if err != nil {
		errorLogger.Fatal(err)
	}
	defer fin.Close()

	dec := gob.NewDecoder(fin)
	if err := dec.Decode(&cacheHosts); err != nil {
		errorLogger.Fatal(err)
	}
}

func ResolvingDial(network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	final_addr, ok := cacheHosts[host]
	if !ok || strings.HasPrefix(final_addr, "err:") {
		return nil, errors.New("No DNS record in my cache")
	}

	errorLogger.Printf("Opening connection: %s!%s", network, final_addr)
	return net.Dial(network, net.JoinHostPort(final_addr, port))
}

var httpClient = http.Client{
	Transport: &http.Transport{Dial: ResolvingDial, DisableKeepAlives: false}}

func DownloadUrl(client http.Client, u url.URL) {
	resp, err := client.Get(u.String())
	if err == nil {
		if resp.StatusCode == http.StatusOK {
			hasher := sha1.New()
			if _, err := io.Copy(hasher, resp.Body); err == nil {
				statusLogger.Printf("%s sha1: %x", u.String(), hasher.Sum(nil))
			} else {
				statusLogger.Printf("%s error: %s", u.String(), err.Error())
			}
		} else {
			statusLogger.Printf("%s status: %d", u.String(), resp.StatusCode)
			if _, err := io.Copy(ioutil.Discard, resp.Body); err != nil {
				errorLogger.Println(err.Error())
			}
		}
		resp.Body.Close()
	} else {
		statusLogger.Printf("%s error: %s\n", u.String(), err.Error())
	}
}

func Downloader(reqCh chan url.URL) {
	defer func() {
		if r := recover(); r != nil {
			errorLogger.Fatalf("Downloader exiting. panic: %v", r)
		}
	}()

	for {
		u, ok := <-reqCh
		if !ok {
			return
		}

		DownloadUrl(httpClient, u)

		//	endCh <- 1
	}
}

func main() {
	flag.Parse()

	readCache()

	fin, err := os.Open(*urlsPath)
	if err != nil {
		errorLogger.Fatal(err)
	}
	defer fin.Close()

	//	waitCh := make(chan int)
	reqCh := make(chan url.URL)

	for i := 0; i < *nDownloaders; i++ {
		go Downloader(reqCh)
	}

	//	ndownloads := 0
	scanner := bufio.NewScanner(fin)
	for scanner.Scan() {
		s := scanner.Text()
		if u, err := url.Parse(s); err == nil {
			if u.Scheme == "https" {
				//				continue
			}
			statusLogger.Printf("%s aaa:", u.String())
			//			select {
			//			case reqCh <- *u:
			//				ndownloads++
			//			case <-waitCh:
			reqCh <- *u
			//				ndownloads--
			//				ndownloads++
			//			}
		} else {
			errorLogger.Printf("%s malformed url: %s", u.String(), err.Error())
		}
	}

	//	for ndownloads > 0 {
	//		<-waitCh
	//		ndownloads--
	//	}
	errorLogger.Println("Should end now")
	select {}
}
