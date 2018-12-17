package config

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func createTempFile(t *testing.T) *os.File {
	file, err := ioutil.TempFile("", "equiv-regs.json")
	if err != nil {
		t.Fatal("failed to create temp file", err)
	}
	return file
}

func TestCreateConfig(t *testing.T) {
	t.Run("no file", func(t *testing.T) {
		cfg, err := CreateConfig("no-such-file")
		if err == nil || cfg != nil {
			t.Error("expected error and nil config")
		}
		if !strings.Contains(err.Error(), "no-such-file") {
			t.Errorf("expected no-such-file; got %s", err)
		}
	})
	t.Run("bad json", func(t *testing.T) {
		file := createTempFile(t)
		defer os.Remove(file.Name())
		content := []byte("rubbish")
		file.Write(content)
		cfg, err := CreateConfig(file.Name())
		if err == nil || cfg != nil {
			t.Error("expected error and nil config")
		}
		if !strings.Contains(err.Error(), "invalid character") {
			t.Errorf("expected invalid char error; got %s", err)
		}
	})
	t.Run("empty json", func(t *testing.T) {
		file := createTempFile(t)
		defer os.Remove(file.Name())
		content := []byte("{}")
		file.Write(content)
		cfg, err := CreateConfig(file.Name())
		if cfg == nil || err != nil {
			t.Error("expected config and nil error")
		}
		if len(cfg.AuthConfigs) != 0 {
			t.Errorf("expected no auth confgs; got %d", len(cfg.AuthConfigs))
		}
	})
	t.Run("valid json", func(t *testing.T) {
		file := createTempFile(t)
		defer os.Remove(file.Name())
		content := []byte("{\"auths\":{\"reg1\": {\"auth\":\"token\"}}}")
		file.Write(content)
		cfg, err := CreateConfig(file.Name())
		if cfg == nil || err != nil {
			t.Error("expected config and nil error")
		}
		if len(cfg.AuthConfigs) != 1 {
			t.Errorf("expected 1 auth config; got %d", len(cfg.AuthConfigs))
		}
		if cfg.AuthConfigs["reg1"].Auth != "token" {
			t.Errorf("expected \"token\" auth token; got %s", cfg.AuthConfigs["reg1"].Auth)
		}
	})
}
