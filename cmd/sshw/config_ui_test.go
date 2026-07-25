package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yinheli/sshw"
)

func TestLoadConfigFileReturnsEmptyForMissingFile(t *testing.T) {
	nodes, err := loadConfigFile(filepath.Join(t.TempDir(), ".sshw.yaml"))
	if err != nil {
		t.Fatalf("expected missing config to load as empty: %v", err)
	}
	if len(nodes) != 0 {
		t.Fatalf("expected empty config, got %d nodes", len(nodes))
	}
}

func TestSaveConfigFileWritesCompactYAML(t *testing.T) {
	file := filepath.Join(t.TempDir(), ".sshw.yaml")
	nodes := []*sshw.Node{
		{
			Name:    "dev",
			Alias:   "dev",
			Host:    "192.168.8.35",
			User:    "appuser",
			KeyPath: "~/.ssh/id_rsa",
		},
	}

	if err := saveConfigFile(file, nodes); err != nil {
		t.Fatalf("save config: %v", err)
	}

	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	content := string(b)
	if !strings.Contains(content, "name: dev") {
		t.Fatalf("expected saved host name, got:\n%s", content)
	}
	if strings.Contains(content, "password:") {
		t.Fatalf("expected empty fields to be omitted, got:\n%s", content)
	}
	if strings.Contains(content, "children:") {
		t.Fatalf("expected plain host to omit children, got:\n%s", content)
	}
}

func TestConfigAPIWritesNestedGroups(t *testing.T) {
	file := filepath.Join(t.TempDir(), ".sshw.yaml")
	payload := map[string]interface{}{
		"nodes": []*yamlNode{
			{
				Name: "production",
				Children: []*yamlNode{
					{Name: "app-1", Host: "192.168.1.2", User: "root"},
				},
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	configAPIHandler(file).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if !strings.Contains(string(content), "children:") {
		t.Fatalf("expected nested children to be preserved, got:\n%s", content)
	}
}

func TestConfigAPIWritesEmptyGroup(t *testing.T) {
	file := filepath.Join(t.TempDir(), ".sshw.yaml")
	payload := map[string]interface{}{
		"nodes": []*yamlNode{
			{Name: "empty group", Children: []*yamlNode{}},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	configAPIHandler(file).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if !strings.Contains(string(content), "children: []") {
		t.Fatalf("expected empty group to keep children marker, got:\n%s", content)
	}
}
