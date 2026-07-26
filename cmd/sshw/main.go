package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/atrox/homedir"
	"github.com/manifoldco/promptui"
	"github.com/yinheli/sshw"
)

const prev = "-parent-"

var (
	Build       = "devel"
	V           = flag.Bool("version", false, "show version")
	VShort      = flag.Bool("v", false, "shorthand for -version")
	H           = flag.Bool("help", false, "show help")
	HShort      = flag.Bool("h", false, "shorthand for -help")
	S           = flag.Bool("s", false, "use local ssh config '~/.ssh/config'")
	Search      = flag.String("search", "", "search hosts by name, alias, user, host, or port")
	SearchShort = flag.String("q", "", "shorthand for -search")
	Config      = flag.Bool("config", false, "open web editor for ~/.sshw.yaml")
	ConfigShort = flag.Bool("c", false, "shorthand for -config")
	ConfigFile  = flag.String("config-file", "~/.sshw.yaml", "config file used by -config")
	ConfigAddr  = flag.String("config-addr", "127.0.0.1:7899", "address used by -config")

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
	os.Args = normalizeSearchArgs(os.Args)
	flag.Parse()
	if !flag.Parsed() {
		flag.Usage()
		return
	}

	if *H || *HShort {
		flag.Usage()
		return
	}

	if *V || *VShort {
		fmt.Println("sshw - ssh client wrapper for automatic login")
		fmt.Println("  git version:", Build)
		fmt.Println("  go version :", runtime.Version())
		return
	}

	if *Config || *ConfigShort {
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

	if query, requested := requestedSearch(); requested {
		node := chooseSearch(sshw.GetConfig(), query)
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

func normalizeSearchArgs(args []string) []string {
	normalized := append([]string(nil), args...)
	for index, arg := range normalized {
		if arg != "-search" && arg != "--search" && arg != "-q" {
			continue
		}
		if index == len(normalized)-1 || strings.HasPrefix(normalized[index+1], "-") {
			normalized[index] = arg + "="
		}
	}
	return normalized
}

func requestedSearch() (string, bool) {
	if flagWasSet("search") {
		return *Search, true
	}
	if flagWasSet("q") {
		return *SearchShort, true
	}
	return "", false
}

func flagWasSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func chooseSearch(trees []*sshw.Node, query string) *sshw.Node {
	nodes := searchHosts(trees, "")
	if len(nodes) == 0 {
		fmt.Println("no hosts configured")
		return nil
	}

	return runSearch(nodes, query, os.Stdin, os.Stdout)
}

func runSearch(nodes []*sshw.Node, query string, stdin io.ReadCloser, stdout io.WriteCloser) *sshw.Node {
	prompt := promptui.Select{
		Label:             "select host",
		Items:             nodes,
		Templates:         templates,
		Size:              20,
		HideSelected:      true,
		StartInSearchMode: true,
		Stdin:             prependInput(query, stdin),
		Stdout:            stdout,
		Searcher: func(input string, index int) bool {
			return nodeMatches(nodes[index], input)
		},
	}
	index, _, err := prompt.Run()
	if err != nil {
		return nil
	}
	return nodes[index]
}

type inputReadCloser struct {
	io.Reader
}

func (inputReadCloser) Close() error {
	return nil
}

func prependInput(input string, reader io.Reader) io.ReadCloser {
	return inputReadCloser{
		Reader: io.MultiReader(strings.NewReader(input), reader),
	}
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
