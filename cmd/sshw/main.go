package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/atrox/homedir"
	"github.com/manifoldco/promptui"
	"github.com/yinheli/sshw"
)

const prev = "-parent-"

var (
	Build      = "devel"
	V          = flag.Bool("version", false, "show version")
	H          = flag.Bool("help", false, "show help")
	S          = flag.Bool("s", false, "use local ssh config '~/.ssh/config'")
	Search     = flag.String("search", "", "search hosts by name, alias, user, host, or port")
	Config     = flag.Bool("config", false, "open web editor for ~/.sshw.yaml")
	ConfigFile = flag.String("config-file", "~/.sshw.yaml", "config file used by -config")
	ConfigAddr = flag.String("config-addr", "127.0.0.1:7899", "address used by -config")

	log = sshw.GetLogger()

	templates = &promptui.SelectTemplates{
		Label:    "✨ {{ . | green}}",
		Active:   "➤ {{ .Name | cyan  }}{{if .Alias}}({{.Alias | yellow}}){{end}} {{if .Host}}{{if .User}}{{.User | faint}}{{`@` | faint}}{{end}}{{.Host | faint}}{{end}}",
		Inactive: "  {{.Name | faint}}{{if .Alias}}({{.Alias | faint}}){{end}} {{if .Host}}{{if .User}}{{.User | faint}}{{`@` | faint}}{{end}}{{.Host | faint}}{{end}}",
	}
)

func findAlias(nodes []*sshw.Node, nodeAlias string) *sshw.Node {
	for _, node := range nodes {
		if node.Alias == nodeAlias {
			return node
		}
		if len(node.Children) > 0 {
			return findAlias(node.Children, nodeAlias)
		}
	}
	return nil
}

func main() {
	flag.Parse()
	if !flag.Parsed() {
		flag.Usage()
		return
	}

	if *H {
		flag.Usage()
		return
	}

	if *V {
		fmt.Println("sshw - ssh client wrapper for automatic login")
		fmt.Println("  git version:", Build)
		fmt.Println("  go version :", runtime.Version())
		return
	}

	if *Config {
		configFile, err := homedir.Expand(*ConfigFile)
		if err != nil {
			log.Error("config error", err)
			os.Exit(1)
		}
		if err := runConfigUI(configFile, *ConfigAddr); err != nil {
			log.Error("config error", err)
			os.Exit(1)
		}
		return
	}

	if *S {
		err := sshw.LoadSshConfig()
		if err != nil {
			log.Error("load ssh config error", err)
			os.Exit(1)
		}
	} else {
		err := sshw.LoadConfig()
		if err != nil {
			log.Error("load config error", err)
			os.Exit(1)
		}
	}

	if *Search != "" {
		node := chooseSearch(sshw.GetConfig(), *Search)
		if node == nil {
			return
		}
		client := sshw.NewClient(node)
		client.Login()
		return
	}

	// login by alias
	if len(os.Args) > 1 {
		var nodeAlias = os.Args[1]
		var nodes = sshw.GetConfig()
		var node = findAlias(nodes, nodeAlias)
		if node != nil {
			client := sshw.NewClient(node)
			client.Login()
			return
		}
	}

	node := choose(nil, sshw.GetConfig())
	if node == nil {
		return
	}

	client := sshw.NewClient(node)
	client.Login()
}

func chooseSearch(trees []*sshw.Node, query string) *sshw.Node {
	nodes := searchHosts(trees, query)
	if len(nodes) == 0 {
		fmt.Printf("no hosts found matching %q\n", query)
		return nil
	}
	return choose(nil, nodes)
}

func searchHosts(nodes []*sshw.Node, query string) []*sshw.Node {
	var results []*sshw.Node
	for _, node := range nodes {
		if len(node.Children) > 0 {
			results = append(results, searchHosts(node.Children, query)...)
			continue
		}
		if nodeMatches(node, query) {
			results = append(results, node)
		}
	}
	return results
}

func nodeMatches(node *sshw.Node, input string) bool {
	content := strings.ToLower(nodeSearchContent(node))
	for _, key := range strings.Fields(strings.ToLower(input)) {
		if !fuzzyContains(content, key) {
			return false
		}
	}
	return true
}

func fuzzyContains(content, query string) bool {
	if query == "" {
		return true
	}
	if strings.Contains(content, query) {
		return true
	}

	contentRunes := []rune(content)
	queryRunes := []rune(query)
	queryIndex := 0
	for _, r := range contentRunes {
		if r == queryRunes[queryIndex] {
			queryIndex++
			if queryIndex == len(queryRunes) {
				return true
			}
		}
	}
	return false
}

func nodeSearchContent(node *sshw.Node) string {
	return fmt.Sprintf("%s %s %s %s %d", node.Name, node.Alias, node.User, node.Host, node.Port)
}

func choose(parent, trees []*sshw.Node) *sshw.Node {
	prompt := promptui.Select{
		Label:        "select host",
		Items:        trees,
		Templates:    templates,
		Size:         20,
		HideSelected: true,
		Searcher: func(input string, index int) bool {
			return nodeMatches(trees[index], input)
		},
	}
	index, _, err := prompt.Run()
	if err != nil {
		return nil
	}

	node := trees[index]
	if len(node.Children) > 0 {
		first := node.Children[0]
		if first.Name != prev {
			first = &sshw.Node{Name: prev}
			node.Children = append(node.Children[:0], append([]*sshw.Node{first}, node.Children...)...)
		}
		return choose(trees, node.Children)
	}

	if node.Name == prev {
		if parent == nil {
			return choose(nil, sshw.GetConfig())
		}
		return choose(nil, parent)
	}

	return node
}
