
package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/golang/groupcache"
)

var host = flag.String("host", "localhost", "peer host")
var port = flag.String("port", "40000", "peer port")

var Fortunes = groupcache.NewGroup("Fortunes", 8 << 20, groupcache.GetterFunc(
	func(ctx groupcache.Context, key string, dest groupcache.Sink) error {
		val := computeFortune(key)
		dest.SetString(val)
		return nil
	}))


func computeFortune(key string) string {
	log.Println("computing value for", key)
	return "fortune cookie"
}


func main() {
	flag.Parse()

	me := "http://" + net.JoinHostPort(*host, *port)
	log.SetPrefix(me + " ")
	peers := groupcache.NewHTTPPool(me)
	peers.Set("http://localhost:40000", "http://localhost:40001")

	log.Println("I am", me)
	go http.ListenAndServe(net.JoinHostPort(*host, *port), nil)

	// give all peers time to start
	time.Sleep(3 * time.Second)

	key, val := "1", ""
	if err := Fortunes.Get(nil, key, groupcache.StringSink(&val)); err == nil {
		log.Println("the value of", key, "is", val)
	}  else {
		log.Println("failed to get key", key, ":", err)
	}
}
