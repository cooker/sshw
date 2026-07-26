package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
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

func TestSaveConfigFileCreatesMissingParentDirectories(t *testing.T) {
	file := filepath.Join(t.TempDir(), "nested", "config", ".sshw.yaml")
	nodes := []*sshw.Node{
		{Name: "dev", Host: "192.168.8.35"},
	}

	if err := saveConfigFile(file, nodes); err != nil {
		t.Fatalf("expected save to create parent directories: %v", err)
	}

	info, err := os.Stat(file)
	if err != nil {
		t.Fatalf("stat saved config: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatalf("expected config mode 0600, got %04o", info.Mode().Perm())
	}
}

func TestSaveConfigFileReplacesReadOnlyFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), ".sshw.yaml")
	if err := os.WriteFile(file, []byte("- name: old\n  host: 127.0.0.1\n"), 0400); err != nil {
		t.Fatalf("write read-only fixture: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(file, 0600)
	})

	nodes := []*sshw.Node{{Name: "new", Host: "127.0.0.2"}}
	if err := saveConfigFile(file, nodes); err != nil {
		t.Fatalf("save over read-only config: %v", err)
	}

	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	if !strings.Contains(string(content), "name: new") {
		t.Fatalf("expected new config content, got:\n%s", content)
	}
}

func TestSaveConfigFileRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "outside.yaml")
	file := filepath.Join(dir, ".sshw.yaml")
	original := []byte("do not overwrite\n")
	if err := os.WriteFile(target, original, 0600); err != nil {
		t.Fatalf("write symlink target: %v", err)
	}
	if err := os.Symlink(target, file); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}

	err := saveConfigFile(file, []*sshw.Node{{Name: "dev", Host: "127.0.0.1"}})
	if err == nil {
		t.Fatal("expected symlink config path to be rejected")
	}
	content, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatalf("read symlink target: %v", readErr)
	}
	if !bytes.Equal(content, original) {
		t.Fatalf("expected symlink target to remain unchanged, got %q", content)
	}
}

func TestSaveConfigFilePreservesOriginalWhenWriteFails(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, ".sshw.yaml")
	original := []byte("- name: original\n  host: 127.0.0.1\n")
	if err := os.WriteFile(file, original, 0600); err != nil {
		t.Fatalf("write original config: %v", err)
	}

	injectedErr := errors.New("injected partial write failure")
	err := saveConfigFileWithWriter(
		file,
		[]*sshw.Node{{Name: "replacement", Host: "127.0.0.2"}},
		func(temp *os.File, content []byte) error {
			if _, writeErr := temp.Write(content[:len(content)/2]); writeErr != nil {
				return writeErr
			}
			return injectedErr
		},
	)
	if !errors.Is(err, injectedErr) {
		t.Fatalf("expected injected write error, got %v", err)
	}

	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read original config: %v", err)
	}
	if !bytes.Equal(content, original) {
		t.Fatalf("expected original config to remain unchanged, got %q", content)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".*.tmp-*"))
	if err != nil {
		t.Fatalf("find temporary configs: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected temporary config cleanup, got %#v", matches)
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

func TestConfigAPIRejectsNullCallback(t *testing.T) {
	file := filepath.Join(t.TempDir(), ".sshw.yaml")
	body := strings.NewReader(`{
		"nodes": [{
			"name": "dev",
			"host": "127.0.0.1",
			"callbackShells": [null]
		}]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/config", body)
	rec := httptest.NewRecorder()

	configAPIHandler(file).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "callback") {
		t.Fatalf("expected callback validation error, got %s", rec.Body.String())
	}
}

func TestConfigAPIRejectsInvalidNodes(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "missing nodes",
			body: `{}`,
			want: "nodes must be an array",
		},
		{
			name: "null node",
			body: `{"nodes":[null]}`,
			want: "nodes[0] must not be null",
		},
		{
			name: "port out of range",
			body: `{"nodes":[{"name":"dev","host":"127.0.0.1","port":65536}]}`,
			want: "port must be between 0 and 65535",
		},
		{
			name: "negative callback delay",
			body: `{"nodes":[{"name":"dev","host":"127.0.0.1","callbackShells":[{"cmd":"pwd","delay":-1}]}]}`,
			want: "delay must not be negative",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), ".sshw.yaml")
			req := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(test.body))
			rec := httptest.NewRecorder()

			configAPIHandler(file).ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), test.want) {
				t.Fatalf("expected error containing %q, got %s", test.want, rec.Body.String())
			}
		})
	}
}

func TestConfigAPICanSaveEmptyNodeList(t *testing.T) {
	file := filepath.Join(t.TempDir(), ".sshw.yaml")
	req := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(`{"nodes":[]}`))
	rec := httptest.NewRecorder()

	configAPIHandler(file).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	nodes, err := loadConfigFile(file)
	if err != nil {
		t.Fatalf("load empty config: %v", err)
	}
	if len(nodes) != 0 {
		t.Fatalf("expected empty saved config, got %d nodes", len(nodes))
	}
}

func TestConfigHTMLValidatesIntegerFields(t *testing.T) {
	expected := []string{
		`name="port" type="number"`,
		`name="jump-port" type="number"`,
		`delayInput.type = "number"`,
		`Number.isSafeInteger`,
		`validity.badInput`,
	}
	for _, fragment := range expected {
		if !strings.Contains(configPageHTML, fragment) {
			t.Fatalf("expected config UI to contain %q", fragment)
		}
	}
}

func TestConfigHTMLProvidesGroupSearchAndServerManagement(t *testing.T) {
	expected := []string{
		`id="group-tree"`,
		`id="search"`,
		`id="server-list"`,
		`id="add-group"`,
		`id="add-subgroup"`,
		`id="add-server"`,
		`id="duplicate-server"`,
		`name="parent"`,
		`function moveNode`,
		`beforeunload`,
		`event.key.toLowerCase() === "s"`,
	}
	for _, fragment := range expected {
		if !strings.Contains(configPageHTML, fragment) {
			t.Fatalf("expected rewritten config UI to contain %q", fragment)
		}
	}
}

func TestConfigHTMLInternationalizationCatalog(t *testing.T) {
	const startTag = `<script id="i18n-catalog" type="application/json">`
	start := strings.Index(configPageHTML, startTag)
	if start < 0 {
		t.Fatal("expected config UI to embed an i18n catalog")
	}
	start += len(startTag)
	end := strings.Index(configPageHTML[start:], "</script>")
	if end < 0 {
		t.Fatal("expected i18n catalog script to have a closing tag")
	}

	var catalog map[string]map[string]string
	if err := json.Unmarshal([]byte(configPageHTML[start:start+end]), &catalog); err != nil {
		t.Fatalf("parse i18n catalog: %v", err)
	}

	expectedLocales := []string{"en", "zh-CN", "ja", "ko", "vi"}
	english := catalog["en"]
	if len(english) == 0 {
		t.Fatal("expected English source translations")
	}
	for _, locale := range expectedLocales {
		translations, ok := catalog[locale]
		if !ok {
			t.Fatalf("expected locale %q", locale)
		}
		if len(translations) != len(english) {
			t.Fatalf("expected locale %q to have %d keys, got %d", locale, len(english), len(translations))
		}
		for key := range english {
			if strings.TrimSpace(translations[key]) == "" {
				t.Fatalf("expected locale %q to define non-empty key %q", locale, key)
			}
		}
	}

	expectedHTML := []string{
		`id="locale"`,
		`data-i18n="action.save"`,
		`data-i18n-placeholder="search.placeholder"`,
		`data-i18n-aria-label="search.label"`,
		`localStorage.setItem("sshw.config.locale"`,
		`document.documentElement.lang = currentLocale`,
	}
	for _, fragment := range expectedHTML {
		if !strings.Contains(configPageHTML, fragment) {
			t.Fatalf("expected internationalized config UI to contain %q", fragment)
		}
	}
}

func TestConfigUIURLUsesLoopbackForWildcardListeners(t *testing.T) {
	tests := []struct {
		addr string
		want string
	}{
		{addr: "127.0.0.1:7899", want: "http://127.0.0.1:7899"},
		{addr: "0.0.0.0:7899", want: "http://127.0.0.1:7899"},
		{addr: "[::]:7899", want: "http://127.0.0.1:7899"},
		{addr: "[::1]:7899", want: "http://[::1]:7899"},
	}

	for _, test := range tests {
		t.Run(test.addr, func(t *testing.T) {
			got := configUIURL(testAddr(test.addr))
			if got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
}

func TestBrowserCommand(t *testing.T) {
	const url = "http://127.0.0.1:7899"
	tests := []struct {
		goos     string
		wantName string
		wantArgs []string
	}{
		{goos: "darwin", wantName: "open", wantArgs: []string{url}},
		{goos: "linux", wantName: "xdg-open", wantArgs: []string{url}},
		{goos: "freebsd", wantName: "xdg-open", wantArgs: []string{url}},
		{goos: "windows", wantName: "rundll32", wantArgs: []string{"url.dll,FileProtocolHandler", url}},
	}

	for _, test := range tests {
		t.Run(test.goos, func(t *testing.T) {
			name, args, err := browserCommand(test.goos, url)
			if err != nil {
				t.Fatalf("browser command: %v", err)
			}
			if name != test.wantName {
				t.Fatalf("expected command %q, got %q", test.wantName, name)
			}
			if strings.Join(args, "\x00") != strings.Join(test.wantArgs, "\x00") {
				t.Fatalf("expected args %#v, got %#v", test.wantArgs, args)
			}
		})
	}

	if _, _, err := browserCommand("plan9", url); err == nil {
		t.Fatal("expected unsupported platform error")
	}
}

type testAddr string

func (a testAddr) Network() string {
	return "tcp"
}

func (a testAddr) String() string {
	return string(a)
}
