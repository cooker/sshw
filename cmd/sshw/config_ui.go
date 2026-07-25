package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/yinheli/sshw"
	"gopkg.in/yaml.v2"
)

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

	url := "http://" + listener.Addr().String()
	fmt.Printf("sshw config UI: %s\n", url)
	fmt.Printf("editing: %s\n", file)
	fmt.Println("press Ctrl+C to stop")

	return http.Serve(listener, mux)
}

func configIndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(configHTML))
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
	out, err := yaml.Marshal(toYAMLNodes(nodes))
	if err != nil {
		return err
	}
	if err := os.WriteFile(file, out, 0600); err != nil {
		return err
	}
	return nil
}

func toYAMLNodes(nodes []*sshw.Node) []*yamlNode {
	out := make([]*yamlNode, 0, len(nodes))
	for _, node := range nodes {
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
			out.CallbackShells = append(out.CallbackShells, &sshw.CallbackShell{
				Cmd:   callback.Cmd,
				Delay: callback.Delay,
			})
		}
	}
	return out
}

const configHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>sshw config</title>
  <style>
    :root {
      color-scheme: light;
      --bg: #f6f7f9;
      --panel: #ffffff;
      --text: #18202b;
      --muted: #657184;
      --line: #d9dee7;
      --primary: #1967d2;
      --primary-dark: #124b9c;
      --danger: #b42318;
      --ok: #1b7f43;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      background: var(--bg);
      color: var(--text);
      font: 14px/1.45 -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 16px;
      padding: 18px 24px;
      background: #111827;
      color: #fff;
    }
    header h1 { margin: 0; font-size: 20px; letter-spacing: 0; }
    header code { color: #c7d2fe; }
    main {
      display: grid;
      grid-template-columns: minmax(260px, 360px) minmax(0, 1fr);
      min-height: calc(100vh - 66px);
    }
    aside {
      border-right: 1px solid var(--line);
      background: var(--panel);
      padding: 18px;
      overflow: auto;
    }
    section {
      padding: 22px;
      overflow: auto;
    }
    .toolbar { display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 14px; }
    button {
      border: 1px solid var(--line);
      border-radius: 6px;
      background: #fff;
      color: var(--text);
      padding: 8px 11px;
      font: inherit;
      cursor: pointer;
    }
    button:hover { border-color: #aab4c3; }
    button.primary {
      border-color: var(--primary);
      background: var(--primary);
      color: #fff;
    }
    button.primary:hover { background: var(--primary-dark); }
    button.danger { color: var(--danger); }
    .host-list { display: grid; gap: 8px; }
    .host-item {
      border: 1px solid var(--line);
      border-radius: 8px;
      background: #fff;
      padding: 10px;
      text-align: left;
      width: 100%;
    }
    .host-item.active { border-color: var(--primary); box-shadow: inset 3px 0 0 var(--primary); }
    .host-name { font-weight: 700; }
    .host-meta { color: var(--muted); font-size: 12px; margin-top: 4px; overflow-wrap: anywhere; }
    .empty { color: var(--muted); border: 1px dashed var(--line); border-radius: 8px; padding: 18px; background: #fff; }
    form {
      max-width: 980px;
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 18px;
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 14px;
    }
    label { display: grid; gap: 6px; color: var(--muted); font-weight: 600; }
    input {
      width: 100%;
      border: 1px solid var(--line);
      border-radius: 6px;
      padding: 9px 10px;
      color: var(--text);
      font: inherit;
      background: #fff;
    }
    input:focus { outline: 2px solid #cfe0ff; border-color: var(--primary); }
    h2 { margin: 0 0 14px; font-size: 22px; }
    h3 { margin: 22px 0 10px; font-size: 16px; }
    .row-list { display: grid; gap: 10px; }
    .row {
      display: grid;
      grid-template-columns: 1fr 150px auto;
      gap: 10px;
      align-items: end;
    }
    .jump {
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 14px;
      background: #fbfcfe;
    }
    .status { color: var(--muted); min-height: 20px; }
    .status.ok { color: var(--ok); }
    .status.error { color: var(--danger); }
    @media (max-width: 760px) {
      main { grid-template-columns: 1fr; }
      aside { border-right: 0; border-bottom: 1px solid var(--line); }
      .grid, .row { grid-template-columns: 1fr; }
      header { align-items: flex-start; flex-direction: column; }
    }
  </style>
</head>
<body>
  <header>
    <h1>sshw config</h1>
    <div>Editing <code id="file-name">.sshw.yaml</code></div>
  </header>
  <main>
    <aside>
      <div class="toolbar">
        <button class="primary" id="add-host" type="button">Add Host</button>
        <button id="add-group" type="button">Add Group</button>
        <button id="add-child-host" type="button">Add Host In Group</button>
        <button id="save" type="button">Save</button>
      </div>
      <div class="status" id="status"></div>
      <div class="host-list" id="host-list"></div>
    </aside>
    <section>
      <div id="editor-empty" class="empty">Select a host or group, or add a new one.</div>
      <form id="group-editor" hidden>
        <h2 id="group-title">Group</h2>
        <div class="grid">
          <label>Name <input name="group-name" required></label>
        </div>
        <div class="toolbar" style="margin-top: 18px">
          <button class="primary" id="apply-group" type="button">Apply Group</button>
          <button class="danger" id="delete-group" type="button">Delete Group</button>
        </div>
      </form>
      <form id="editor" hidden>
        <h2 id="editor-title">Host</h2>
        <div class="grid">
          <label>Name <input name="name" required></label>
          <label>Alias <input name="alias"></label>
          <label>Host <input name="host" required></label>
          <label>User <input name="user" placeholder="root"></label>
          <label>Port <input name="port" inputmode="numeric" placeholder="22"></label>
          <label>Key path <input name="keypath" placeholder="~/.ssh/id_rsa"></label>
          <label>Agent path <input name="agentpath"></label>
          <label>Passphrase <input name="passphrase" type="password"></label>
          <label>Password <input name="password" type="password"></label>
        </div>

        <h3>Jump host</h3>
        <div class="toolbar">
          <button id="toggle-jump" type="button">Add Jump Host</button>
        </div>
        <div class="jump" id="jump-fields" hidden>
          <div class="grid">
            <label>Name <input name="jump-name"></label>
            <label>Host <input name="jump-host"></label>
            <label>User <input name="jump-user"></label>
            <label>Port <input name="jump-port" inputmode="numeric"></label>
            <label>Key path <input name="jump-keypath"></label>
            <label>Password <input name="jump-password" type="password"></label>
          </div>
        </div>

        <h3>Callback commands</h3>
        <div class="toolbar">
          <button id="add-callback" type="button">Add Command</button>
        </div>
        <div class="row-list" id="callbacks"></div>

        <div class="toolbar" style="margin-top: 18px">
          <button class="primary" id="apply" type="button">Apply Changes</button>
          <button class="danger" id="delete-host" type="button">Delete Host</button>
        </div>
      </form>
    </section>
  </main>
  <script>
    let nodes = [];
    let selectedPath = null;
    let selectedKind = "";
    let hasJump = false;
    const $ = (id) => document.getElementById(id);

    function blankHost() {
      return { name: "", alias: "", host: "", user: "", port: 0, keypath: "", agentpath: "", passphrase: "", password: "", jump: [], callbackShells: [] };
    }

    function blankGroup() {
      return { name: "", children: [] };
    }

    function setStatus(message, type = "") {
      const el = $("status");
      el.textContent = message;
      el.className = "status " + type;
    }

    function normalizeHost(host) {
      const hasChildren = Array.isArray(host.children);
      return {
        name: host.name || "",
        alias: host.alias || "",
        host: host.host || "",
        user: host.user || "",
        port: Number(host.port || 0),
        keypath: host.keypath || "",
        agentpath: host.agentpath || "",
        passphrase: host.passphrase || "",
        password: host.password || "",
        jump: Array.isArray(host.jump) ? host.jump : [],
        callbackShells: Array.isArray(host.callbackShells) ? host.callbackShells : [],
        ...(hasChildren ? { children: host.children.map(normalizeHost) } : {})
      };
    }

    function nodeRefs(list = nodes, depth = 0, basePath = []) {
      const refs = [];
      list.forEach((host, index) => {
        const path = basePath.concat(index);
        const isGroup = Array.isArray(host.children);
        refs.push({ host, path, depth, kind: isGroup ? "group" : "host" });
        if (host.children && host.children.length) {
          refs.push(...nodeRefs(host.children, depth + 1, path));
        }
      });
      return refs;
    }

    function samePath(a, b) {
      return Array.isArray(a) && Array.isArray(b) && a.length === b.length && a.every((value, index) => value === b[index]);
    }

    function getHost(path) {
      let current = nodes;
      for (let i = 0; i < path.length; i++) {
        const host = current[path[i]];
        if (i === path.length - 1) return host;
        current = host.children || [];
      }
      return null;
    }

    function setHost(path, host) {
      const parent = getParentList(path);
      parent[path[path.length - 1]] = host;
    }

    function deleteHost(path) {
      const parent = getParentList(path);
      parent.splice(path[path.length - 1], 1);
    }

    function getParentList(path) {
      let current = nodes;
      for (let i = 0; i < path.length - 1; i++) {
        current = current[path[i]].children || [];
      }
      return current;
    }

    async function loadConfig() {
      const res = await fetch("/api/config");
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "failed to load config");
      $("file-name").textContent = data.file;
      nodes = (data.nodes || []).map(normalizeHost);
      renderList();
      setStatus("Loaded", "ok");
    }

    async function saveConfig() {
      applyEditor(false);
      applyGroupEditor(false);
      const res = await fetch("/api/config", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ nodes })
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "failed to save config");
      setStatus("Saved", "ok");
    }

    function renderList() {
      const list = $("host-list");
      list.innerHTML = "";
      const refs = nodeRefs();
      if (refs.length === 0) {
        list.innerHTML = '<div class="empty">No hosts or groups configured.</div>';
        showEmpty();
        return;
      }
      refs.forEach((ref) => {
        const host = ref.host;
        const item = document.createElement("button");
        item.type = "button";
        item.className = "host-item" + (samePath(ref.path, selectedPath) ? " active" : "");
        item.style.paddingLeft = (10 + ref.depth * 18) + "px";
        item.innerHTML = '<div class="host-name"></div><div class="host-meta"></div>';
        item.querySelector(".host-name").textContent = (ref.kind === "group" ? "Group: " : "Host: ") + (host.name || "(unnamed)");
        item.querySelector(".host-meta").textContent = ref.kind === "group" ? ((host.children || []).length + " item(s)") : [host.alias && "(" + host.alias + ")", host.user && host.user + "@", host.host, host.port || ""].filter(Boolean).join(" ");
        item.addEventListener("click", () => selectNode(ref.path));
        list.appendChild(item);
      });
    }

    function selectNode(path) {
      applyEditor(false);
      applyGroupEditor(false);
      selectedPath = path;
      const node = normalizeHost(getHost(path));
      if (Array.isArray(node.children)) {
        selectGroup(path, node);
        return;
      }
      selectHost(path, node);
    }

    function selectGroup(path, group) {
      selectedKind = "group";
      $("editor-empty").hidden = true;
      $("editor").hidden = true;
      $("group-editor").hidden = false;
      $("group-title").textContent = group.name || "Group";
      $("group-editor").elements["group-name"].value = group.name || "";
      renderList();
    }

    function selectHost(path, host) {
      selectedKind = "host";
      $("editor-empty").hidden = true;
      $("editor").hidden = false;
      $("group-editor").hidden = true;
      $("editor-title").textContent = host.name || "Host";
      for (const key of ["name", "alias", "host", "user", "port", "keypath", "agentpath", "passphrase", "password"]) {
        $("editor").elements[key].value = host[key] || "";
      }
      hasJump = host.jump.length > 0;
      setJumpVisible(hasJump);
      const jump = normalizeHost(host.jump[0] || {});
      $("editor").elements["jump-name"].value = jump.name || "";
      $("editor").elements["jump-host"].value = jump.host || "";
      $("editor").elements["jump-user"].value = jump.user || "";
      $("editor").elements["jump-port"].value = jump.port || "";
      $("editor").elements["jump-keypath"].value = jump.keypath || "";
      $("editor").elements["jump-password"].value = jump.password || "";
      renderCallbacks(host.callbackShells);
      renderList();
    }

    function showEmpty() {
      selectedPath = null;
      selectedKind = "";
      $("editor").hidden = true;
      $("group-editor").hidden = true;
      $("editor-empty").hidden = false;
    }

    function readEditor() {
      const form = $("editor");
      const host = blankHost();
      for (const key of ["name", "alias", "host", "user", "keypath", "agentpath", "passphrase", "password"]) {
        host[key] = form.elements[key].value.trim();
      }
      host.port = Number(form.elements.port.value || 0);
      if (hasJump) {
        const jump = blankHost();
        jump.name = form.elements["jump-name"].value.trim();
        jump.host = form.elements["jump-host"].value.trim();
        jump.user = form.elements["jump-user"].value.trim();
        jump.port = Number(form.elements["jump-port"].value || 0);
        jump.keypath = form.elements["jump-keypath"].value.trim();
        jump.password = form.elements["jump-password"].value.trim();
        if (jump.host) host.jump = [jump];
      }
      host.callbackShells = Array.from(document.querySelectorAll(".callback-row")).map((row) => ({
        cmd: row.querySelector('[data-field="cmd"]').value.trim(),
        delay: Number(row.querySelector('[data-field="delay"]').value || 0)
      })).filter((row) => row.cmd);
      return host;
    }

    function applyEditor(showMessage = true) {
      if (!selectedPath || selectedKind !== "host" || $("editor").hidden) return;
      const host = readEditor();
      setHost(selectedPath, host);
      renderList();
      if (showMessage) setStatus("Applied locally. Save to write the file.", "ok");
    }

    function applyGroupEditor(showMessage = true) {
      if (!selectedPath || selectedKind !== "group" || $("group-editor").hidden) return;
      const current = normalizeHost(getHost(selectedPath));
      current.name = $("group-editor").elements["group-name"].value.trim();
      current.host = "";
      current.alias = "";
      current.user = "";
      current.port = 0;
      current.keypath = "";
      current.agentpath = "";
      current.passphrase = "";
      current.password = "";
      current.jump = [];
      current.callbackShells = [];
      current.children = current.children || [];
      setHost(selectedPath, current);
      renderList();
      if (showMessage) setStatus("Group applied locally. Save to write the file.", "ok");
    }

    function selectedGroupChildren() {
      if (!selectedPath || selectedKind !== "group") return nodes;
      const group = getHost(selectedPath);
      group.children = group.children || [];
      return group.children;
    }

    function setJumpVisible(visible) {
      hasJump = visible;
      $("jump-fields").hidden = !visible;
      $("toggle-jump").textContent = visible ? "Remove Jump Host" : "Add Jump Host";
    }

    function renderCallbacks(callbacks) {
      const root = $("callbacks");
      root.innerHTML = "";
      (callbacks || []).forEach(addCallbackRow);
    }

    function addCallbackRow(callback = { cmd: "", delay: 0 }) {
      const root = $("callbacks");
      const row = document.createElement("div");
      row.className = "row callback-row";
      row.innerHTML = '<label>Command <input data-field="cmd"></label><label>Delay ms <input data-field="delay" inputmode="numeric"></label><button class="danger" type="button">Remove</button>';
      row.querySelector('[data-field="cmd"]').value = callback.cmd || "";
      row.querySelector('[data-field="delay"]').value = callback.delay || "";
      row.querySelector("button").addEventListener("click", () => row.remove());
      root.appendChild(row);
    }

    $("add-host").addEventListener("click", () => {
      const list = selectedGroupChildren();
      list.push(blankHost());
      const path = selectedKind === "group" ? selectedPath.concat(list.length - 1) : [nodes.length - 1];
      selectNode(path);
      setStatus("New host added locally. Save to write the file.", "ok");
    });
    $("add-group").addEventListener("click", () => {
      const list = selectedGroupChildren();
      list.push(blankGroup());
      const path = selectedKind === "group" ? selectedPath.concat(list.length - 1) : [nodes.length - 1];
      selectNode(path);
      setStatus("New group added locally. Save to write the file.", "ok");
    });
    $("add-child-host").addEventListener("click", () => {
      if (selectedKind !== "group") {
        setStatus("Select a group first.", "error");
        return;
      }
      const list = selectedGroupChildren();
      list.push(blankHost());
      selectNode(selectedPath.concat(list.length - 1));
      setStatus("New host added to group locally. Save to write the file.", "ok");
    });
    $("apply").addEventListener("click", () => applyEditor(true));
    $("apply-group").addEventListener("click", () => applyGroupEditor(true));
    $("save").addEventListener("click", () => saveConfig().catch((err) => setStatus(err.message, "error")));
    $("delete-host").addEventListener("click", () => {
      if (!selectedPath) return;
      deleteHost(selectedPath);
      showEmpty();
      renderList();
      setStatus("Host deleted locally. Save to write the file.", "ok");
    });
    $("delete-group").addEventListener("click", () => {
      if (!selectedPath) return;
      deleteHost(selectedPath);
      showEmpty();
      renderList();
      setStatus("Group deleted locally. Save to write the file.", "ok");
    });
    $("toggle-jump").addEventListener("click", () => setJumpVisible(!hasJump));
    $("add-callback").addEventListener("click", () => addCallbackRow());

    loadConfig().catch((err) => setStatus(err.message, "error"));
  </script>
</body>
</html>`
