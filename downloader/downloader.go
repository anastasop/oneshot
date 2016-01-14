package main

import (
	"bufio"
	"crypto/sha1"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
)

var finPath = flag.String("fin", "urls4.txt", "urls file path")
var nDnsResolvers = flag.Int("wdns", 4, "number of dns workers")

type DnsCache struct {
	sync.Mutex
	hosts map[string]string
}

type DnsResponse struct {
	addr string
	err  error
}

type DnsRequest struct {
	host  string
	respc chan DnsResponse
}

type Downloaders struct {
	sync.Mutex
	chans map[string]chan string
}

var downloadersForHosts = Downloaders{chans: make(map[string]chan string)}

var defaultDnsCache = DnsCache{hosts: make(map[string]string)}
var dnsResolveChannel = make(chan DnsRequest)

func DnsResolver(c chan DnsRequest) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("dns resolve failed: %v", r)
		}
	}()
	for {
		req, ok := <-c
		if !ok {
			return
		}

		defaultDnsCache.Lock()
		addr, ok := defaultDnsCache.hosts[req.host]
		defaultDnsCache.Unlock()

		if ok {
			req.respc <- DnsResponse{addr, nil}
		} else {
			addrs, err := net.LookupHost(req.host)
			if err == nil {
				if len(addrs) > 0 {
					defaultDnsCache.Lock()
					defaultDnsCache.hosts[req.host] = addrs[0]
					defaultDnsCache.Unlock()
					log.Println("Resolved", req.host, "to", addrs[0])
					req.respc <- DnsResponse{addr: addrs[0]}
				} else {
					req.respc <- DnsResponse{err: errors.New("Empty DNS response: " + req.host)}
				}
			} else {
				log.Println("Resolving: ", req.host, " failed", err.Error())
				req.respc <- DnsResponse{err: err}
			}
		}
	}
}

func ResolvingDial(network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	log.Println("ResolvingDial:", host, port)
	var final_addr = host
	if ip_addr := net.ParseIP(host); ip_addr == nil {
		req := DnsRequest{host: host, respc: make(chan DnsResponse)}
		dnsResolveChannel <- req
		resp := <-req.respc

		if resp.err != nil {
			return nil, resp.err
		} else {
			final_addr = resp.addr
		}
	}
	log.Println("ResolvingDial Result:", network, final_addr)

	log.Printf("Opening connection: %s!%s", network, final_addr)
	return net.Dial(network, net.JoinHostPort(final_addr, port))
}

func main() {
	log.SetFlags(log.Lshortfile)
	flag.Parse()

	defer close(dnsResolveChannel)
	for i := 0; i < *nDnsResolvers; i++ {
		go DnsResolver(dnsResolveChannel)
	}

	fin, err := os.Open(*finPath)
	if err != nil {
		log.Fatal(err)
	}
	defer fin.Close()

	waitCh := make(chan int)

	ndownloads := 0
	scanner := bufio.NewScanner(fin)
	for scanner.Scan() {
		s := scanner.Text()
		if u, err := url.Parse(s); err == nil {
			if u.Scheme == "https" {
				continue
			}
			downloadersForHosts.Lock()
			downloader, ok := downloadersForHosts.chans[u.Host]
			downloadersForHosts.Unlock()
			if !ok {
				downloader = DownloaderForHost(u.Host, waitCh)
				downloadersForHosts.Lock()
				downloadersForHosts.chans[u.Host] = downloader
				downloadersForHosts.Unlock()
			}
			ndownloads++
			//			cli := http.Client{Transport: &http.Transport{Dial: ResolvingDial}}
			//			DownloadUrl(cli, s)
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

func DownloadUrl(client http.Client, url string) {
	log.Println("Get: ", url)
	resp, err := client.Get(url)
	if err == nil {
		if resp.StatusCode == http.StatusOK {
			hasher := sha1.New()
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
}

func DownloaderForHost(host string, endCh chan int) chan string {
	client := http.Client{Transport: &http.Transport{Dial: ResolvingDial}}

	c := make(chan string)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("downloader panic: %v", r)
				downloadersForHosts.Lock()
				delete(downloadersForHosts.chans, host)
				downloadersForHosts.Unlock()
			}
		}()
		url, ok := <-c
		if !ok {
			return
		}

		DownloadUrl(client, url)

		endCh <- 1
	}()
	return c
}
