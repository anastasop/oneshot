
package main

import (
	"crypto/sha1"
	"hash"
	"hash/crc32"
	"io/ioutil"
	"flag"
	"fmt"
	"os"
)

var hashfunc = flag.String("f", "", "hash function to use")
var printheader = flag.Bool("p", false, "print the hash type with the value, crc32:af8d3ac5")

func main() {
	flag.Parse()

	var hasher hash.Hash
	switch *hashfunc {
	case "crc32":
		hasher = crc32.NewIEEE()
	case "sha1":
		hasher = sha1.New()
	default:
		fmt.Fprintf(os.Stderr, "hash %s not supported. Use one of crc32, sha1\n", *hashfunc)
		os.Exit(2)
	}

	var stdinData []byte
	for _, arg := range flag.Args() {
		if arg == "-" {
			if stdinData == nil {
				var err error
				stdinData, err = ioutil.ReadAll(os.Stdin)
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to read stdin: %s", err)
					os.Exit(2)
				}
			}
			hasher.Write(stdinData)
		} else {
			hasher.Write([]byte(arg))
		}
	}

	if *printheader {
		fmt.Printf("%s:%x\n", *hashfunc, hasher.Sum(nil))
	} else {
		fmt.Printf("%x\n", hasher.Sum(nil))
	}
	
}
