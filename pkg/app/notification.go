// pkg/app/notification.go
package app

import (
	"fmt"
	"os/exec"
)

func showMacNotification(title, message string) error {
	script := fmt.Sprintf(`display notification "%s" with title "%s"`, message, title)
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}
