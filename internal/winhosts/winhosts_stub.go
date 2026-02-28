//go:build !windows

package winhosts

import "fmt"

const (
	SystemHostsPath = `C:\Windows\System32\drivers\etc\hosts`
	StartMarker     = "# >>> Zephy Managed Start"
	EndMarker       = "# <<< Zephy Managed End"
)

func IsAdmin() (bool, error) {
	return false, fmt.Errorf("admin check not supported on this platform")
}

func ApplyManagedBlock(lines []string, flushDNS bool) error {
	return fmt.Errorf("not supported on this platform")
}

func RemoveManagedBlock(flushDNS bool) error {
	return fmt.Errorf("not supported on this platform")
}

