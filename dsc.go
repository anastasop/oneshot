package main

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"flag"
	"fmt"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

var (
	errStat     = errors.New("Stat")
	errOpen     = errors.New("Open")
	errRead     = errors.New("Read")
	errExif     = errors.New("Exif")
	errExifTime = errors.New("ExifTime")
)

type photo struct {
	path     string
	sha1     string
	mimeType string
	width    int
	height   int
	size     int64
	exifTime time.Time
	err      error
}

func newPhoto(path string, info os.FileInfo, buf *bytes.Buffer) *photo {
	p := new(photo)
	p.path = path

	if info == nil {
		p.err = errStat
		return p
	} else {
		p.size = info.Size()
	}
	buf.Grow(int(info.Size()))

	fin, err := os.Open(path)
	if err != nil {
		p.err = errOpen
		return p
	}
	defer fin.Close()

	_, err = buf.ReadFrom(fin)
	if err != nil {
		p.err = errRead
		return p
	}
	p.mimeType = strings.TrimSpace(strings.Split(http.DetectContentType(buf.Bytes()), ";")[0])

	r := bytes.NewReader(buf.Bytes())

	h := sha1.New()
	if _, err := r.WriteTo(h); err != nil {
		panic(err)
	}
	p.sha1 = fmt.Sprintf("%x", h.Sum(nil))

	if _, err := r.Seek(0, io.SeekStart); err != nil {
		panic(err)
	}
	if ex, err := exif.Decode(r); err == nil {
		if m, err := ex.DateTime(); err == nil {
			p.exifTime = m
		} else if _, ok := err.(exif.TagNotPresentError); !ok {
			p.exifTime = time.Now()
		} else {
			p.exifTime = time.Now().UTC()
		}
	} else {
		p.err = errExif
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		panic(err)
	}

	if cfg, err := jpeg.DecodeConfig(r); err == nil {
		p.width = cfg.Width
		p.height = cfg.Height
	}

	return p
}

var photos = make(map[string]*photo)

func copyPhoto(p *photo, buf *bytes.Buffer) {
	t := p.exifTime
	dir := filepath.Join(*dst, fmt.Sprintf("%04d-%02d", t.Year(), int(t.Month())))

	if err := os.MkdirAll(dir, 0755); err != nil {
		panic(err)
	}

	fname := filepath.Join(dir, fmt.Sprintf("IMG_%04d%02d%02dT%02d%02d%02d.jpg", t.Year(), int(t.Month()), t.Day(), t.Hour(), t.Minute(), t.Second()))
	if err := ioutil.WriteFile(fname, buf.Bytes(), 0644); err != nil {
		panic(err)
	}
}

func emitPhoto(p *photo, buf *bytes.Buffer, doReport, doCopy bool) {
	if p.err != nil {
		fmt.Printf("Error: %s: %v\n", p.path, p.err)
	} else if curr, present := photos[p.sha1]; !present {
		photos[p.sha1] = p
		if doReport {
			fmt.Printf("Photo: %s %s %dx%d %d %s\n", p.exifTime.Format("2006-01-02 15:04:05"), p.mimeType, p.width, p.height, p.size, p.path)
		}
		if doCopy {
			copyPhoto(p, buf)
		}
	} else if doReport {
		fmt.Printf("Double: %s of %s\n", p.path, curr.path)
	}
}

func scanDir(root string, doReport, doCopy bool) error {
	var buf bytes.Buffer

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Error: %s: %v\n", path, err)
			return nil
		}

		if info == nil && err == nil {
			log.Fatal("path %s: info is nil but err is not", path)
		}

		if _, elem := filepath.Split(path); elem != "" {
			// Skip "hidden" files or directories.
			if elem[0] == '.' {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if info.Mode().IsRegular() {
			if re.MatchString(info.Name()) {
				buf.Reset()
				emitPhoto(newPhoto(path, info, &buf), &buf, doReport, doCopy)
			}
		}

		return nil
	})

	return err
}

var src = flag.String("s", ".", "dir to scan")
var dst = flag.String("d", ".", "dir for copies")
var reStr = flag.String("r", "(?i)^(IMG|DSC|XRS).*JPG$", "regex for file names")
var doCopy = flag.Bool("c", false, "do copy")
var re *regexp.Regexp

func main() {
	log.SetPrefix("")
	log.SetFlags(0)
	flag.Parse()

	re = regexp.MustCompile(*reStr)

	if err := scanDir(*dst, false, false); err != nil {
		log.Fatal(err)
	}

	if err := scanDir(*src, true, *doCopy); err != nil {
		log.Fatal(err)
	}
}
