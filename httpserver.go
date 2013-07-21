
package main

import (
	"flag"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sort"
	"strconv"
)

var port = flag.Int("p", 40000, "server port")
var size = flag.Int("s", 1024, "size of the web page")

type ByteSlice []byte
func (b ByteSlice) Len() int { return len(b) }
func (b ByteSlice) Less(i, j int) bool { return b[i] < b[j] }
func (b ByteSlice) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

func garbageHandler(w http.ResponseWriter, r *http.Request) {
	var b ByteSlice = make([]byte, 1024)
	for i := 0; i < len(b); i++ {
		b[i] = byte(rand.Intn(256))
	}
	sort.Sort(b)
	w.Header().Add("Content-Type", "application/octet-stream")
	w.Write(b)
}

func main() {
	flag.Parse()

	http.HandleFunc("/", garbageHandler)
	log.Println("server listens at", *port)
	err := http.ListenAndServe(net.JoinHostPort("", strconv.Itoa(*port)), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
