package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Profile struct {
	Name      string `json:"name"`
	ListenIP  string `json:"listen_ip"`
	Port      int    `json:"port"`
	HostsFile string `json:"hosts_file"`
}

type File struct {
	Profiles []Profile `json:"profiles"`
}

func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg File
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if len(cfg.Profiles) == 0 {
		return nil, fmt.Errorf("config has no profiles")
	}

	seen := map[string]struct{}{}
	for i, p := range cfg.Profiles {
		if p.Name == "" {
			return nil, fmt.Errorf("profile[%d]: name is required", i)
		}
		if p.ListenIP == "" {
			return nil, fmt.Errorf("profile[%s]: listen_ip is required", p.Name)
		}
		if p.Port <= 0 || p.Port > 65535 {
			return nil, fmt.Errorf("profile[%s]: invalid port", p.Name)
		}
		if p.HostsFile == "" {
			return nil, fmt.Errorf("profile[%s]: hosts_file is required", p.Name)
		}

		key := fmt.Sprintf("%s:%d", p.ListenIP, p.Port)
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("duplicate listen address: %s", key)
		}
		seen[key] = struct{}{}
	}

	return &cfg, nil
}
