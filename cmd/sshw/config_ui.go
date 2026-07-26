package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/yinheli/sshw"
	"gopkg.in/yaml.v2"
)

//go:embed config_ui.html
var configPageHTML string

type yamlNode struct {
	Name           string           `json:"name" yaml:"name,omitempty"`
	Alias          string           `json:"alias" yaml:"alias,omitempty"`
	Host           string           `json:"host" yaml:"host,omitempty"`
	User           string           `json:"user" yaml:"user,omitempty"`
	Port           int              `json:"port" yaml:"port,omitempty"`
	KeyPath        string           `json:"keypath" yaml:"keypath,omitempty"`
	AgentPath      string           `json:"agentpath" yaml:"agentpath,omitempty"`
	Passphrase     string           `json:"passphrase" yaml:"passphrase,omitempty"`
	Password       string           `json:"password" yaml:"password,omitempty"`
	CallbackShells []*CallbackShell `json:"callbackShells" yaml:"callback-shells,omitempty"`
	Children       []*yamlNode      `json:"children" yaml:"children,omitempty"`
	Jump           []*yamlNode      `json:"jump" yaml:"jump,omitempty"`
}

type CallbackShell struct {
	Cmd   string        `json:"cmd" yaml:"cmd,omitempty"`
	Delay time.Duration `json:"delay" yaml:"delay,omitempty"`
}

func (n *yamlNode) MarshalYAML() (interface{}, error) {
	out := yaml.MapSlice{}
	addString := func(key, value string) {
		if value != "" {
			out = append(out, yaml.MapItem{Key: key, Value: value})
		}
	}
	addString("name", n.Name)
	addString("alias", n.Alias)
	addString("host", n.Host)
	addString("user", n.User)
	if n.Port > 0 {
		out = append(out, yaml.MapItem{Key: "port", Value: n.Port})
	}
	addString("keypath", n.KeyPath)
	addString("agentpath", n.AgentPath)
	addString("passphrase", n.Passphrase)
	addString("password", n.Password)
	if len(n.CallbackShells) > 0 {
		out = append(out, yaml.MapItem{Key: "callback-shells", Value: n.CallbackShells})
	}
	if n.Children != nil {
		out = append(out, yaml.MapItem{Key: "children", Value: n.Children})
	}
	if len(n.Jump) > 0 {
		out = append(out, yaml.MapItem{Key: "jump", Value: n.Jump})
	}
	return out, nil
}

func runConfigUI(file, addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", configIndexHandler)
	mux.HandleFunc("/api/config", configAPIHandler(file))

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	url := configUIURL(listener.Addr())
	fmt.Printf("sshw config UI: %s\n", url)
	fmt.Printf("editing: %s\n", file)
	fmt.Println("press Ctrl+C to stop")
	if err := openBrowser(url); err != nil {
		fmt.Printf("could not open browser: %v\n", err)
	}

	return http.Serve(listener, mux)
}

func configUIURL(addr net.Addr) string {
	host, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return "http://" + addr.String()
	}
	if host == "" {
		host = "127.0.0.1"
	} else if ip := net.ParseIP(host); ip != nil && ip.IsUnspecified() {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port)
}

func openBrowser(url string) error {
	name, args, err := browserCommand(runtime.GOOS, url)
	if err != nil {
		return err
	}

	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		_ = cmd.Wait()
	}()
	return nil
}

func browserCommand(goos, url string) (string, []string, error) {
	switch goos {
	case "darwin":
		return "open", []string{url}, nil
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}, nil
	case "linux", "freebsd", "openbsd", "netbsd", "solaris":
		return "xdg-open", []string{url}, nil
	default:
		return "", nil, fmt.Errorf("unsupported platform %q; open %s manually", goos, url)
	}
}

func configIndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(configPageHTML))
}

func configAPIHandler(file string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			nodes, err := loadConfigFile(file)
			if err != nil {
				writeConfigError(w, http.StatusInternalServerError, err)
				return
			}
			writeConfigJSON(w, map[string]interface{}{
				"file":  file,
				"nodes": toYAMLNodes(nodes),
			})
		case http.MethodPost:
			var payload struct {
				Nodes []*yamlNode `json:"nodes"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				writeConfigError(w, http.StatusBadRequest, err)
				return
			}
			if err := validateYAMLNodes(payload.Nodes, "nodes"); err != nil {
				writeConfigError(w, http.StatusBadRequest, err)
				return
			}
			nodes := fromYAMLNodes(payload.Nodes)
			if err := saveConfigFile(file, nodes); err != nil {
				writeConfigError(w, http.StatusInternalServerError, err)
				return
			}
			writeConfigJSON(w, map[string]string{"status": "saved"})
		default:
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func validateYAMLNodes(nodes []*yamlNode, path string) error {
	if nodes == nil {
		return fmt.Errorf("%s must be an array", path)
	}
	for index, node := range nodes {
		nodePath := fmt.Sprintf("%s[%d]", path, index)
		if node == nil {
			return fmt.Errorf("%s must not be null", nodePath)
		}
		if node.Port < 0 || node.Port > 65535 {
			return fmt.Errorf("%s.port must be between 0 and 65535", nodePath)
		}
		for callbackIndex, callback := range node.CallbackShells {
			if callback == nil {
				return fmt.Errorf("%s.callbackShells[%d] must not be null", nodePath, callbackIndex)
			}
			if callback.Delay < 0 {
				return fmt.Errorf("%s.callbackShells[%d].delay must not be negative", nodePath, callbackIndex)
			}
		}
		if node.Children != nil {
			if err := validateYAMLNodes(node.Children, nodePath+".children"); err != nil {
				return err
			}
		}
		if node.Jump != nil {
			if err := validateYAMLNodes(node.Jump, nodePath+".jump"); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeConfigJSON(w http.ResponseWriter, value interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(value)
}

func writeConfigError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func loadConfigFile(file string) ([]*sshw.Node, error) {
	b, err := os.ReadFile(file)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return nil, nil
	}

	var nodes []*sshw.Node
	if err := yaml.Unmarshal(b, &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

func saveConfigFile(file string, nodes []*sshw.Node) error {
	return saveConfigFileWithWriter(file, nodes, func(temp *os.File, content []byte) error {
		_, err := temp.Write(content)
		return err
	})
}

func saveConfigFileWithWriter(file string, nodes []*sshw.Node, write func(*os.File, []byte) error) error {
	out, err := yaml.Marshal(toYAMLNodes(nodes))
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	dir := filepath.Dir(file)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config directory %q: %w", dir, err)
	}
	if info, err := os.Lstat(file); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to replace symlink config %q", file)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect config %q: %w", file, err)
	}

	temp, err := os.CreateTemp(dir, ".sshw.tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary config in %q: %w", dir, err)
	}
	tempName := temp.Name()
	keepTemp := true
	defer func() {
		if keepTemp {
			_ = os.Remove(tempName)
		}
	}()

	if err := temp.Chmod(0600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("set temporary config permissions %q: %w", tempName, err)
	}
	if err := write(temp, out); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write temporary config %q: %w", tempName, err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync temporary config %q: %w", tempName, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary config %q: %w", tempName, err)
	}
	if err := replaceConfigFile(tempName, file); err != nil {
		return fmt.Errorf("replace config %q: %w", file, err)
	}
	keepTemp = false
	return nil
}

func replaceConfigFile(source, destination string) error {
	err := os.Rename(source, destination)
	if err == nil || runtime.GOOS != "windows" {
		return err
	}

	info, statErr := os.Stat(destination)
	if statErr != nil {
		return err
	}
	originalMode := info.Mode().Perm()
	if chmodErr := os.Chmod(destination, 0600); chmodErr != nil {
		return err
	}
	if retryErr := os.Rename(source, destination); retryErr != nil {
		_ = os.Chmod(destination, originalMode)
		return retryErr
	}
	return nil
}

func toYAMLNodes(nodes []*sshw.Node) []*yamlNode {
	out := make([]*yamlNode, 0, len(nodes))
	for _, node := range nodes {
		if node == nil {
			continue
		}
		out = append(out, toYAMLNode(node))
	}
	return out
}

func toYAMLNode(node *sshw.Node) *yamlNode {
	if node == nil {
		return nil
	}
	out := &yamlNode{
		Name:       node.Name,
		Alias:      node.Alias,
		Host:       node.Host,
		User:       node.User,
		Port:       node.Port,
		KeyPath:    node.KeyPath,
		AgentPath:  node.AgentPath,
		Passphrase: node.Passphrase,
		Password:   node.Password,
	}
	if node.Children != nil {
		out.Children = toYAMLNodes(node.Children)
	}
	if len(node.Jump) > 0 {
		out.Jump = toYAMLNodes(node.Jump)
	}
	if len(node.CallbackShells) > 0 {
		out.CallbackShells = make([]*CallbackShell, 0, len(node.CallbackShells))
		for _, callback := range node.CallbackShells {
			if callback == nil {
				continue
			}
			out.CallbackShells = append(out.CallbackShells, &CallbackShell{
				Cmd:   callback.Cmd,
				Delay: callback.Delay,
			})
		}
	}
	return out
}

func fromYAMLNodes(nodes []*yamlNode) []*sshw.Node {
	out := make([]*sshw.Node, 0, len(nodes))
	for _, node := range nodes {
		if node == nil {
			continue
		}
		out = append(out, fromYAMLNode(node))
	}
	return out
}

func fromYAMLNode(node *yamlNode) *sshw.Node {
	if node == nil {
		return nil
	}
	out := &sshw.Node{
		Name:       node.Name,
		Alias:      node.Alias,
		Host:       node.Host,
		User:       node.User,
		Port:       node.Port,
		KeyPath:    node.KeyPath,
		AgentPath:  node.AgentPath,
		Passphrase: node.Passphrase,
		Password:   node.Password,
	}
	if node.Children != nil {
		out.Children = fromYAMLNodes(node.Children)
	}
	if len(node.Jump) > 0 {
		out.Jump = fromYAMLNodes(node.Jump)
	}
	if len(node.CallbackShells) > 0 {
		out.CallbackShells = make([]*sshw.CallbackShell, 0, len(node.CallbackShells))
		for _, callback := range node.CallbackShells {
			if callback == nil {
				continue
			}
			out.CallbackShells = append(out.CallbackShells, &sshw.CallbackShell{
				Cmd:   callback.Cmd,
				Delay: callback.Delay,
			})
		}
	}
	return out
}
