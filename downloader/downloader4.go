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
)

func download(client http.Client, u *url.URL) (hash []byte, rerr error) {
	url := u.String()

	defer func() {
		if r := recover(); r != nil {
			rerr = fmt.Errorf("download %s panic %#v", url, r)
		}
	}()

	resp, err := client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("download %s roundtripper %s", url, err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		hasher := sha1.New()
		if _, err := io.Copy(hasher, resp.Body); err == nil {
			return hasher.Sum(nil), nil
		} else {
			return nil, fmt.Errorf("download %s sha1 %s", url, err.Error())
		}
	} else {
		_, err := io.Copy(ioutil.Discard, resp.Body)
		if err == nil {
			return nil, fmt.Errorf("download %s http %d", url, resp.StatusCode)
		} else {
			return nil, fmt.Errorf("download %s http %d copy %s", url, resp.StatusCode, err.Error())
		}
	}
}

func downloader(client http.Client, c chan *url.URL) {
	for {
		u, ok := <-c
		if !ok {
			log.Println("Exiting")
			return
		}

		hash, err := download(client, u)
		if err == nil {
			log.Printf("%s sha1 %x", u.String(), hash)
		} else {
			log.Printf("%s err %s", u.String(), err.Error())
		}
	}
}

var urlsPath = flag.String("urls", "urls.txt", "urls file path")
var nDownloaders = flag.Int("nd", 4, "number of downloader")

func main() {
	flag.Parse()

	cli := http.Client{}
	reqCh := make(chan *url.URL)

	for i := 0; i < *nDownloaders; i++ {
		go downloader(cli, reqCh)
	}

	fin, err := os.Open(*urlsPath)
	if err != nil {
		log.Fatal(err)
	}
	defer fin.Close()

	scanner := bufio.NewScanner(fin)
	for scanner.Scan() {
		s := scanner.Text()
		if u, err := url.Parse(s); err == nil {
			reqCh <- u
		} else {
			log.Printf("%s malformed %s", s, err.Error())
		}
	}
}
