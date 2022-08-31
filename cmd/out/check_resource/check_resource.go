package check_resource

import (
	"fmt"
	"github.com/springernature/halfpipe-deploy-resource/logger"
	"io"
	"os"
	"path"
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

func CheckResource(args []string, l logger.CapturingWriter) {
	baseDir := args[1]
	l.Println("Making sure we are running with the latest built resource")
	builtWithRef := readFile(l, "/opt/resource/builtWithRef")
	currentRef := readFile(l, path.Join(baseDir, "git/.git/ref"))
	l.Println(fmt.Sprintf("Built with ref '%s', current ref '%s'", builtWithRef, currentRef))

	if builtWithRef != currentRef {
		panic("Running test with old docker image..Thats no good...")
	}
}
