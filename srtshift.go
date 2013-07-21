/* reads and srt file from stdin,
   shifts the subtitles of frame by some msec
   and writes the new srt to stdin
*/

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	shiftmsec  int = 0
	afterframe int = 0
)

func main() {
	flag.IntVar(&shiftmsec, "shift", 0, "msec to shift")
	flag.IntVar(&afterframe, "frame", 0, "shift only frame >=")
	flag.Parse()

	r := bufio.NewReader(os.Stdin)
	numre := regexp.MustCompile("^[0-9]+$")
	currframe := 0
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)

		if len(line) > 0 && numre.MatchString(line) {
			currframe, _ = strconv.Atoi(line)
		}

		if strings.Index(line, "-->") >= 0 && currframe >= afterframe {
			fields := strings.Fields(line)

			var hour, min, sec, msec int
			fmt.Sscanf(fields[0], "%02d:%02d:%02d,%03d", &hour, &min, &sec, &msec)
			fromd := time.Date(2012, time.June, 4, hour, min, sec, int(time.Duration(msec)*time.Millisecond), time.UTC)
			fromd = fromd.Add(time.Duration(shiftmsec) * time.Millisecond)
			fmt.Sscanf(fields[2], "%02d:%02d:%02d,%03d", &hour, &min, &sec, &msec)
			tod := time.Date(2012, time.June, 4, hour, min, sec, int(time.Duration(msec)*time.Millisecond), time.UTC)
			tod = tod.Add(time.Duration(shiftmsec) * time.Millisecond)

			line = fmt.Sprintf("%02d:%02d:%02d,%03d --> %02d:%02d:%02d,%03d",
				fromd.Hour(), fromd.Minute(), fromd.Second(), int(time.Duration(fromd.Nanosecond())/time.Millisecond),
				tod.Hour(), tod.Minute(), tod.Second(), int(time.Duration(tod.Nanosecond())/time.Millisecond))
		}
		fmt.Println(line)
	}
}
