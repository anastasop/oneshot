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

var defaultDnsCache = DnsCache{hosts: make(map[string]string)}
var dnsResolveChannel = make(chan DnsRequest)

func DnsResolver(c chan DnsRequest) {
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
				defaultDnsCache.Lock()
				defaultDnsCache.hosts[req.host] = addrs[0]
				defaultDnsCache.Unlock()
				log.Println("Resolved", req.host, "to", addrs[0])
				req.respc <- DnsResponse{addr: addrs[0]}
			} else {
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

	var final_addr = host
	if ip_addr := net.ParseIP(host); ip_addr == nil {
		req := DnsRequest{host: host, respc: make(chan DnsResponse)}
		dnsResolveChannel <- req
		resp := <-req.respc

		if resp.err != nil {
			return nil, err
		} else {
			final_addr = resp.addr
		}
	}

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

func DownloaderForHost(host string, endCh chan int) chan string {
	client := http.Client{Transport: &http.Transport{Dial: ResolvingDial}}
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
