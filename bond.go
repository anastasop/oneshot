package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/garyburd/go-oauth/oauth"
)

const user_timeline_url = "https://api.twitter.com/1.1/statuses/home_timeline.json"

// flags
var (
	refresh_duration = flag.Int("i", 60, "refresh internal in minutes")
	tweets_per_page  = flag.Int("p", 20, "tweets per page")
)

// configuration
var (
	user_credentials oauth.Credentials
	oauth_client     oauth.Client
)

type Timeline struct {
	RawJson   []byte
	TweetsIds []struct {
		Id int64
	}
}

func main() {
	flag.Parse()

	creds_json, err := ioutil.ReadFile("./credentials.json")
	if err != nil {
		log.Fatal(err)
	}
	creds_map := make(map[string]string)
	err = json.Unmarshal(creds_json, &creds_map)
	if err != nil {
		log.Fatal(err)
	}
	user_credentials.Token = creds_map["userOAuthToken"]
	user_credentials.Secret = creds_map["userOAuthSecret"]
	oauth_client = oauth.Client{
		oauth.Credentials{
			Token:  creds_map["applicationOAuthToken"],
			Secret: creds_map["applicationOAuthSecret"],
		},
		// not used since we are already have a token
		// but included for reference
		"https://api.twitter.com/oauth/request_token",
		"https://api.twitter.com/oauth/authorize",
		"https://api.twitter.com/oauth/access_token",
	}

	c := time.Tick(time.Duration(*refresh_duration) * time.Minute)
	iterateTimeline(c)
}

func writeTimelinePageToFile(t *Timeline) {
	fname := fmt.Sprintf("tweets/%d.json", time.Now().Unix())
	if err := ioutil.WriteFile(fname, t.RawJson, os.FileMode(0666)); err != nil {
		log.Println("Failed to write file:", err)
	}
}

func extractCursorsFromPage(t *Timeline) (max_id, since_id int64) {
	since_id = t.TweetsIds[0].Id
	max_id = t.TweetsIds[0].Id
	for _, tweet := range t.TweetsIds {
		if tweet.Id > since_id {
			since_id = tweet.Id
		}
		if tweet.Id < max_id {
			max_id = tweet.Id
		}
	}
	max_id--
	return
}

func iterateTimeline(c <-chan time.Time) {
	// first bootstrap
	page := getTimelinePage(*tweets_per_page, 0, 0)
	if page == nil || len(page.TweetsIds) == 0 {
		log.Fatal("Cannot bootstrap the timeline")
	}
	log.Println("timeline bootstrapped with", len(page.TweetsIds), "tweets")
	writeTimelinePageToFile(page)

	_, since_id := extractCursorsFromPage(page)
	for {
		<-c // wait next tick
		max_id, next_since_id, cand_since_id := int64(0), since_id, since_id
		for {
			page := getTimelinePage(*tweets_per_page, since_id, max_id)
			if page == nil || len(page.TweetsIds) == 0 {
				log.Println("no page this time. Waiting for next tick")
				break
			}

			log.Println("got page with", len(page.TweetsIds), "tweets")
			writeTimelinePageToFile(page)

			// prepare next iteration
			max_id, cand_since_id = extractCursorsFromPage(page)
			if cand_since_id > next_since_id {
				next_since_id = cand_since_id
			}

			if len(page.TweetsIds) < *tweets_per_page {
				log.Println("got fewer tweets than expected(", *tweets_per_page, "). Waiting for next tick")
				break
			}
		}
		since_id = next_since_id
	}
}

func getTimelinePage(count int, since_id, max_id int64) *Timeline {
	log.Println("requesting new timeline page")
	v := url.Values{}
	if count > 0 {
		v.Set("count", strconv.FormatInt(int64(count), 10))
	}
	if since_id > 0 {
		v.Set("since_id", strconv.FormatInt(since_id, 10))
	}
	if max_id > 0 {
		v.Set("max_id", strconv.FormatInt(max_id, 10))
	}

	var timeline Timeline

	if resp, err := oauth_client.Get(http.DefaultClient, &user_credentials, user_timeline_url, v); err == nil {
		defer resp.Body.Close()
		timeline.RawJson, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("Failed to read response body:", err)
			return nil
		}
		if resp.StatusCode != http.StatusOK {
			log.Println("Response Status is not 200 but", resp.StatusCode, ":error:", string(timeline.RawJson))
			log.Println(resp.Header)
			return nil
		}
	} else {
		log.Println("Get failed:", err)
		return nil
	}

	if err := json.Unmarshal(timeline.RawJson, &timeline.TweetsIds); err != nil {
		log.Println("Failed to parse json response:", err)
		return nil
	}

	return &timeline
}
