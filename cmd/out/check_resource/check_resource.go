package check_resource

import (
	"fmt"
	"github.com/springernature/halfpipe-deploy-resource/logger"
	"io"
	"os"
	"strings"
)

func readFile(l logger.CapturingWriter, path string) string {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	builtWithRef, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(string(builtWithRef))

}

func CheckResource(l logger.CapturingWriter) {
	l.Println("Making sure we are running with the latest built resource")
	builtWithRef := readFile(l, "/opt/resource/builtWithRef")
	currentRef := readFile(l, "git/.git/ref")
	l.Println(fmt.Sprintf("Build with ref '%s', current ref '%s'", builtWithRef, currentRef))

	if builtWithRef != currentRef {
		panic("Running test with old docker image..Thats no good...")
	}
}
