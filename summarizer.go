package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var page_url = flag.String("u", "", "url to summarize")

func printSummary(
	hdr string,
	sel *goquery.Selection,
	summary func(s *goquery.Selection) (string, bool)) {

	fmt.Println(hdr)
	sel.Each(func(i int, s *goquery.Selection) {
		if sum, dis := summary(s); dis {
			fmt.Println("\t", sum)
		}
	})
	fmt.Println("")
}

func printText(s *goquery.Selection) (string, bool) {
	return s.Text(), true
}

func main() {
	flag.Parse()

	site_url, err := url.Parse(*page_url)
	if err != nil {
		log.Fatal(err)
	}

	doc, err := goquery.NewDocument(*page_url)
	if err != nil {
		log.Fatal(err)
	}

	printSummary("title", doc.Find("title"), printText)
	printSummary("meta", doc.Find("meta"), func(s *goquery.Selection) (string, bool) {
		name, exists := s.Attr("name")
		if !exists {
			name, _ = s.Attr("property")
		}
		content, _ := s.Attr("content")
		return fmt.Sprintf("%s: %s", name, content), true
	})
	printSummary("h1", doc.Find("h1"), printText)
	printSummary("h2", doc.Find("h2"), printText)
	printSummary("a", doc.Find("a"), func(s *goquery.Selection) (string, bool) {
		href, exists := s.Attr("href")
		if exists && strings.HasPrefix(href, "http") {
			if hu, err := url.Parse(href); err == nil && hu.Host != site_url.Host {
				return s.Text() + ": " + href, true
			}
		}
		return "", false
	})
}
