package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type GlobalConfig struct {
	Prompts map[string]string `json:"prompts"`
}

func (c *GlobalConfig) LookupPrompt(key string) string {
	prompt := c.Prompts[key]
	if prompt == "" {
		return key
	}
	return prompt
}

func configDir() string {
	if dir := os.Getenv("MERGEFILES_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".config", "mergefiles")
}

func readOrCreateConfig(conf *GlobalConfig) error {
	dir := configDir()
	path := filepath.Join(dir, "config.json")

	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(filepath.Dir(path), 0o700)
		if err != nil {
			return fmt.Errorf("failed to create config dir: %w", err)
		}
		f, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
		if err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}
		defer func() { _ = f.Close() }()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		err = enc.Encode(conf)
		if err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer func() { _ = f.Close() }()
	err = json.NewDecoder(f).Decode(conf)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	return nil
}

func InitConfig() (GlobalConfig, error) {
	conf := GlobalConfig{
		Prompts: map[string]string{
			"default": `You are an expert software developer. You will always output every file in TOTAL if you changed it. 
Do not output unchanged files. 
You will ask questions to make sure you understood everything perfectly right.
Also ask for further info, like documentation, files etc. if needed.
I provide you with the content of several files. Every file will be introduced by '--- START File: [path] ---' and ended by '--- END FILE ---'

This is your task:


`,
			"go": `You are an expert go software developer. You will always output every file in TOTAL if you changed it. 
Do not output unchanged files. 
You will ask questions to make sure you understood everything perfectly right.
Also ask for further info, like documentation, files etc. if needed.
I provide you with the content of several files. Every file will be introduced by '--- START File: [path] ---' and ended by '--- END FILE ---'

This is your task:

`,
			"shell": "Return a one-line bash command with the functionality I will describe. Return ONLY the command ready to run in the terminal. The command should do the following:\n",
		},
	}
	err := readOrCreateConfig(&conf)
	if err != nil {
		return GlobalConfig{}, err
	}

	return conf, nil
}
