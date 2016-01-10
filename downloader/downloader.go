package main

import (
	"bufio"
	"crypto/sha1"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
)

var finPath = flag.String("fin", "urls4.txt", "urls file path")

func main() {
	log.SetFlags(log.Lshortfile)
	flag.Parse()

	fin, err := os.Open(*finPath)
	if err != nil {
		log.Fatal(err)
	}
	defer fin.Close()

	downloaders := make(map[string]chan string)
	waitCh := make(chan int)

	ndownloads := 0
	scanner := bufio.NewScanner(fin)
	for scanner.Scan() {
		s := scanner.Text()
		if u, err := url.Parse(s); err == nil {
			downloader, ok := downloaders[u.Host]
			if !ok {
				downloader = DownloaderForHost(u.Host, waitCh)
				downloaders[u.Host] = downloader
			}
			ndownloads++
			go func(c chan string) {
				c <- s
			}(downloader)
		} else {
			log.Println(err)
		}
	}

	for ndownloads > 0 {
		<-waitCh
		ndownloads--
	}
}

func LoggingDial(network, addr string) (net.Conn, error) {
	log.Printf("Opening connection: %s!%s", network, addr)
	return net.Dial(network, addr)
}

func DownloaderForHost(host string, endCh chan int) chan string {
	client := http.Client{Transport: &http.Transport{Dial: LoggingDial}}
	hasher := sha1.New()

	c := make(chan string)
	go func() {
		url, ok := <-c
		if !ok {
			return
		}

		resp, err := client.Get(url)
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				hasher.Reset()
				if _, err := io.Copy(hasher, resp.Body); err == nil {
					fmt.Printf("%s sha1:%x\n", url, hasher.Sum(nil))
				} else {
					fmt.Printf("%s errn:%v\n", err)
				}
			} else {
				fmt.Sprintf("%s http:%d", url, resp.StatusCode)
				if _, err := io.Copy(ioutil.Discard, resp.Body); err != nil {
					log.Println(err)
				}
			}
			resp.Body.Close()
		} else {
			log.Println(err)
		}
		endCh <- 1
	}()
	return c
}
