package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

const appName = "integration-test-app"
const testDomain = "public.springernature.app"

var workingDir = os.Args[1]

func main() {
	// Please invoke from halfpipe-cf-plugin root with pwd as first argument.
	// go run .integration_test/integration.go `pwd`
	//

	loginToCF()
	makeSureSpaceIsCleaned()

	// Run 1
	push()
	check()
	promote()
	cleanup()

	// Run 2
	push()
	check()
	promote()
	cleanup()

	// Run 3
	push()
	check()
	promote()
	cleanup()
}

func cleanup() {
	runOutWithCommand("halfpipe-cleanup")
}

func push() {
	runOutWithCommand("halfpipe-push")
}

func check() {
	runOutWithCommand("halfpipe-check")
}

func promote() {
	runOutWithCommand("halfpipe-promote")
}

func runOutWithCommand(command string) {
	fmt.Printf("==== RUNNING WITH OUT COMMAND %s ====\n", command)
	pathToRequest := createRequest(command)

	cat := exec.Command("cat", pathToRequest)
	out := exec.Command("/opt/resource/out", workingDir)

	r, w := io.Pipe()
	cat.Stdout = w
	out.Stdin = r

	out.Stdout = os.Stdout
	out.Stderr = os.Stderr

	if err := cat.Start(); err != nil {
		panic(err)
	}

	if err := out.Start(); err != nil {
		panic(err)
	}

	if err := cat.Wait(); err != nil {
		panic(err)
	}
	w.Close()

	if err := out.Wait(); err != nil {
		panic(err)
	}
}

func loginToCF() {
	fmt.Println("==== LOGGING IN ====")
	login := exec.Command("cf", "login",
		"-a", os.Getenv("CF_API"),
		"-u", os.Getenv("CF_USERNAME"),
		"-p", os.Getenv("CF_PASSWORD"),
		"-o", os.Getenv("CF_ORG"),
		"-s", os.Getenv("CF_SPACE"),
	)

	output, err := login.Output()
	if err != nil {
		fmt.Println(string(output))
		panic(err)
	}
}

func makeSureSpaceIsCleaned() {
	fmt.Println("==== CLEANING SPACE ====")
	for _, app := range getApps() {
		deleteCmd := exec.Command("cf", "delete", app.Name, "-f")
		output, err := deleteCmd.Output()
		if err != nil {
			fmt.Println(string(output))
			panic(err)
		}
	}
}

func getApps() (apps []App) {
	appsCms := exec.Command("cf", "apps")

	var buffer bytes.Buffer
	appsCms.Stdout = &buffer
	appsCms.Stderr = &buffer

	if err := appsCms.Start(); err != nil {
		panic(err)
	}

	err := appsCms.Wait()
	if err != nil {
		fmt.Println(string(buffer.Bytes()))
		panic(err)
	}

	for _, line := range strings.Split(string(buffer.Bytes()), "\n")[4:] {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			if strings.HasPrefix(fields[0], appName) {
				var routes []string
				for _, field := range fields {
					if strings.Contains(field, testDomain) {
						routes = append(routes, strings.Replace(field, ",", "", -1))
					}
				}
				apps = append(apps, App{fields[0], routes})
			}
		}
	}

	return
}

func createRequest(command string) (pathToRequest string) {
	r := Request{
		Source: Source{
			API:      os.Getenv("CF_API"),
			Org:      os.Getenv("CF_ORG"),
			Space:    os.Getenv("CF_SPACE"),
			Username: os.Getenv("CF_USERNAME"),
			Password: os.Getenv("CF_PASSWORD"),
		},
		Params: Params{
			Command:      command,
			ManifestPath: ".integration_test/manifest.yml",
			AppPath:      ".integration_test",
			TestDomain:   testDomain,
			GitRefPath:   ".integration_test/gitRef",
		},
	}

	b, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}

	f, err := ioutil.TempFile("", "")
	if err != nil {
		panic(err)
	}

	_, err = f.Write(b)
	if err != nil {
		panic(err)
	}

	return f.Name()
}

type App struct {
	Name   string
	Routes []string
}

type Request struct {
	Source Source `json:"source"`
	Params Params `json:"params"`
}

type Source struct {
	API      string `json:"api"`
	Org      string `json:"org"`
	Space    string `json:"space"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Params struct {
	Command      string `json:"command"`
	ManifestPath string `json:"manifestPath"`
	AppPath      string `json:"appPath"`
	TestDomain   string `json:"testDomain"`
	GitRefPath   string `json:gitRefPath`
}
