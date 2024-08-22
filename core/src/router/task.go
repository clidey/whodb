package router

import (
	"os"
	"os/exec"
	"runtime"

	"github.com/clidey/whodb/core/src/log"
)

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	default:
		log.Logger.Warnf("Unsupported platform. Please open the URL manually: %s\n", url)
	}
	if err != nil {
		log.Logger.Warnf("Failed to open browser: %v\n", err)
	}
}

func isDocker() bool {
	_, err := os.Stat("/.dockerenv")
	return !os.IsNotExist(err)
}
