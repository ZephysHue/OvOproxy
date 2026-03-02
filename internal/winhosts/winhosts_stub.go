//go:build !windows

package winhosts

import "fmt"

const SystemHostsPath = `/etc/hosts`

func IsAdmin() (bool, error) {
	return false, fmt.Errorf("admin check not supported on this platform")
}

func ApplyProfileBlock(profileId string, lines []string, flushDNS bool) error {
	return fmt.Errorf("not supported on this platform")
}

func RemoveProfileBlock(profileId string, flushDNS bool) error {
	return fmt.Errorf("not supported on this platform")
}

func RemoveAllZephyBlocks(flushDNS bool) error {
	return fmt.Errorf("not supported on this platform")
}

func GetEnabledProfiles() ([]string, error) {
	return nil, fmt.Errorf("not supported on this platform")
}

func ApplyManagedBlock(lines []string, flushDNS bool) error {
	return fmt.Errorf("not supported on this platform")
}

func RemoveManagedBlock(flushDNS bool) error {
	return fmt.Errorf("not supported on this platform")
}

