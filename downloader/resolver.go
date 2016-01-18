package main

import (
	"bufio"
	"encoding/gob"
	"flag"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
)

var urlsFile = flag.String("urls", "urls.txt", "urls file")
var cacheFile = flag.String("cache", "resolutions.gob", "cache file")
var cache map[string]string

func readCache() {
	fin, err := os.Open(*cacheFile)
	if err != nil {
		if perr := err.(*os.PathError); os.IsNotExist(perr.Err) {
			cache = make(map[string]string)
			return
		} else {
			log.Fatal(err)
		}
	}
	defer fin.Close()

	dec := gob.NewDecoder(fin)
	if err := dec.Decode(&cache); err != nil {
		log.Fatal(err)
	}
}

func writeCache() {
	fout, err := os.Create(*cacheFile)
	if err != nil {
		log.Fatal(err)
	}
	defer fout.Close()

	enc := gob.NewEncoder(fout)
	if err := enc.Encode(&cache); err != nil {
		log.Fatal(err)
	}
}

func logCache(logErrors bool) {
	nok, nerr := 0, 0
	for host, addr := range cache {
		if strings.HasPrefix(addr, "err:") {
			nerr++
			if logErrors {
				log.Printf("%s: %s", host, addr)
			}
		} else {
			nok++
		}
	}
	log.Printf("Cache: total: %d  ok: %d  err: %d", nok+nerr, nok, nerr)
}

func main() {
	log.SetFlags(log.Lshortfile)
	flag.Parse()

	if *cacheFile != "" {
		readCache()
	} else {
		cache = make(map[string]string)
	}
	logCache(true)

	fin, err := os.Open(*urlsFile)
	if err != nil {
		log.Fatal(err)
	}
	defer fin.Close()

	scanner := bufio.NewScanner(fin)
	for scanner.Scan() {
		s := scanner.Text()
		if u, err := url.Parse(s); err == nil {
			chost, ok := cache[u.Host]
			if !ok || strings.HasPrefix(chost, "err:") {
				addrs, err := net.LookupHost(u.Host)
				if err != nil {
					log.Println(err)
					cache[u.Host] = "err:" + err.Error()
				} else {
					log.Printf("Resolved %s to %s", u.Host, addrs[0])
					cache[u.Host] = addrs[0]
				}
			}
		} else {
			log.Println(err)
		}
	}

	logCache(true)
	writeCache()
}
