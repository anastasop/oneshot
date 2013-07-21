
package main

import (
	"bufio"
	"fmt"
	"log"
	"io"
	"os"
	"os/exec"
)

const (
	kLogFile = "./teeproc.txt"
	kCmdPath = "/bin/cat"
)

func xfer(r io.Reader, w io.Writer, logger *log.Logger, closeOnExit ...io.Closer) {
	for _, s := range closeOnExit {
		defer s.Close()
	}
	bin := bufio.NewReader(r)
	for {
		line, err := bin.ReadString('\n')
		if err == nil {
			logger.Print(line)
			fmt.Fprint(w, line)
		} else if err == io.EOF {
			break
		} else {
			logger.Fatalln("xfer failed:", err)
		}
	}
}

func main() {
	logfile, err := os.Create(kLogFile)
	if err != nil {
		os.Exit(2)
	}
	defer logfile.Close()

	logger := log.New(logfile, log.Prefix(), log.Flags())

	cmd := exec.Command(kCmdPath)
	pin, e1 := cmd.StdinPipe()
	pout, e2 := cmd.StdoutPipe()
	perr, e3 := cmd.StderrPipe()
	if e1 != nil && e2 != nil && e3 != nil {
		logger.Fatalln("failed to create pipes:", e1, e2, e3)
	}
	if err := cmd.Start(); err != nil {
		logger.Fatalln("failed to start cmd:", err)
	}

	go xfer(os.Stdin, pin, logger, pin)
	go xfer(pout, os.Stdout, logger)
	go xfer(perr, os.Stderr, logger)

	if err := cmd.Wait(); err != nil {
		logger.Fatalln("cmd.Wait():", err)
	}
}
