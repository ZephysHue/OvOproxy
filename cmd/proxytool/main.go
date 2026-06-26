package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"zephy/internal/config"
	"zephy/internal/hosts"
	"zephy/internal/proxymanager"
)

func getExeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func main() {
	defaultConfig := filepath.Join(getExeDir(), "configs", "proxy_profiles.json")
	cfgPath := flag.String("config", defaultConfig, "path to profiles config")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	exeDir := getExeDir()
	manager := proxymanager.New()
	defer manager.StopAll()

	errCh := make(chan error, len(cfg.Profiles))
	for _, p := range cfg.Profiles {
		hostsPath := p.HostsFile
		if !filepath.IsAbs(hostsPath) {
			hostsPath = filepath.Join(exeDir, hostsPath)
		}
		table, err := hosts.Load(hostsPath)
		if err != nil {
			log.Fatalf("load hosts failed for profile=%s: %v", p.Name, err)
		}
		if err := manager.StartProxy(p.Name, p.ListenIP, p.Port, table.GetAll()); err != nil {
			errCh <- err
			continue
		}
	}

	err = <-errCh
	log.Fatalf("proxy server stopped: %v", err)
}
