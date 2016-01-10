package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"os"
	"time"
)

var finPath = flag.String("fin", "urls.txt", "urls file path")
var sampleSize = flag.Int("k", 1, "#samples for each host")

type Sampler struct {
	sample []interface{}
	size   int
	seen   int
}

func NewSampler(size int) *Sampler {
	s := &Sampler{size: size}
	s.sample = make([]interface{}, size)
	return s
}

func (s *Sampler) CheckIn(v interface{}) {
	if s.seen < s.size {
		s.sample[s.seen] = v
		s.seen += 1
	} else {
		s.seen += 1
		i := rand.Intn(s.seen)
		if i < s.size {
			s.sample[i] = v
		}
	}
}

func (s *Sampler) GetSample() ([]interface{}, bool) {
	if s.seen < s.size {
		return s.sample, false
	}
	return s.sample, true
}

func main() {
	rand.Seed(time.Now().Unix())
	log.SetFlags(log.Lshortfile)
	flag.Parse()

	fin, err := os.Open(*finPath)
	if err != nil {
		log.Fatal(err)
	}
	defer fin.Close()

	samplers := make(map[string]*Sampler)

	scanner := bufio.NewScanner(fin)
	for scanner.Scan() {
		s := scanner.Text()
		if u, err := url.Parse(s); err == nil {
			sampler := samplers[u.Host]
			if sampler == nil {
				sampler = NewSampler(*sampleSize)
				samplers[u.Host] = sampler
			}
			sampler.CheckIn(s)
		} else {
			log.Println(err)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	for _, sampler := range samplers {
		if sample, full := sampler.GetSample(); full {
			for _, url := range sample {
				fmt.Println(url)
			}
		}
	}

	log.Printf("Sampled %d hosts", len(samplers))
}
