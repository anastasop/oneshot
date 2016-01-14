package main

import (
	"bufio"
	"crypto/sha1"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
)

var urlsPath = flag.String("urls", "urls.txt", "urls file path")
var nDownloaders = flag.Int("nd", 8, "number of downloaders")

var clientPerHost struct {
	sync.Mutex
	ch map[string]http.Client
}

func DownloadUrl(client http.Client, u url.URL) {
	resp, err := client.Get(u.String())
	if err == nil {
		if resp.StatusCode == http.StatusOK {
			hasher := sha1.New()
			if _, err := io.Copy(hasher, resp.Body); err == nil {
				fmt.Printf("%s sha1: %x\n", u.String(), hasher.Sum(nil))
			} else {
				fmt.Printf("%s err: %v\n", u.String(), err.Error())
			}
		} else {
			fmt.Sprintf("%s http: %d", u, resp.StatusCode)
			if _, err := io.Copy(ioutil.Discard, resp.Body); err != nil {
				log.Println(err.Error())
			}
		}
		resp.Body.Close()
	} else {
		log.Println("Get", u.String(), "error:", err.Error())
	}
}

func Downloader(reqCh chan url.URL, endCh chan int) {
	defer func() {
		if r := recover(); r != nil {
			//clientPerHost.Lock()
			//delete(clientPerHost.ch, host)
			//clientPerHost.Unlock()
			log.Printf("Downloader exiting. panic: %v", r)
		}
	}()

	for {
		u, ok := <-reqCh
		if !ok {
			return
		}

		clientPerHost.Lock()
		client, ok := clientPerHost.ch[u.Host]
		clientPerHost.Unlock()

		if !ok {
			client := http.Client{}
			clientPerHost.Lock()
			clientPerHost.ch[u.Host] = client
			clientPerHost.Unlock()
		}

		DownloadUrl(client, u)

		endCh <- 1
	}
}

func main() {
	log.SetFlags(log.Lshortfile)
	flag.Parse()

	clientPerHost.ch = make(map[string]http.Client)

	fin, err := os.Open(*urlsPath)
	if err != nil {
		log.Fatal(err)
	}
	defer fin.Close()

	waitCh := make(chan int)
	reqCh := make(chan url.URL)

	for i := 0; i < *nDownloaders; i++ {
		go Downloader(reqCh, waitCh)
	}

	ndownloads := 0
	scanner := bufio.NewScanner(fin)
	for scanner.Scan() {
		s := scanner.Text()
		if u, err := url.Parse(s); err == nil {
			if u.Scheme == "https" {
				fmt.Printf("%s ignored", s)
				continue
			}
			select {
			case reqCh <- *u:
				ndownloads++
			case <-waitCh:
				ndownloads--
			}
		} else {
			log.Printf("%s malformed url: %s", err.Error())
		}
	}

	for ndownloads > 0 {
		<-waitCh
		ndownloads--
	}
}
