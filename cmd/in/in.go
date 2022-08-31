package main

import (
	"encoding/json"
	"github.com/springernature/halfpipe-deploy-resource/config"
	"io"
	"log"
	"os"
	"strings"
)

func readFile(l *log.Logger, path string) string {
	f, err := os.Open(path)
	if err != nil {
		l.Panic(err)
	}
	builtWithRef, err := io.ReadAll(f)
	if err != nil {
		l.Panic(err)
	}
	return strings.TrimSpace(string(builtWithRef))

}

func main() {
	// Just echo stdin to stdout.
	logger := log.New(os.Stderr, "", 0)

	inData, err := io.ReadAll(os.Stdin)
	if err != nil {
		logger.Println(err.Error())
		os.Exit(1)
	}

	// This is just to make sure the pipeline where we build and test the resource is correct..
	req := config.Request{}
	err = json.Unmarshal(inData, &req)
	if err != nil {
		logger.Panicf("Failed to unmarshal request %s", err)
	}
	if req.Params.Command == "check" {
		logger.Println("Making sure we are running with the latest built resource")
		builtWithRef := readFile(logger, "/opt/resource/builtWithRef")
		currentRef := readFile(logger, "git/.git/ref")
		logger.Printf("Build with ref '%s', current ref '%s'", builtWithRef, currentRef)

		if builtWithRef != currentRef {
			logger.Panic("Running test with old docker image..Thats no good...")
		}
	}

	logger.SetOutput(os.Stdout)
	logger.Println(string(inData))
}
