package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/manifoldco/promptui"
	"github.com/yinheli/sshw"
)

func TestNodeMatchesCaseInsensitiveFuzzySearch(t *testing.T) {
	node := &sshw.Node{
		Name:  "Production API",
		Alias: "prod-api",
		User:  "deploy",
		Host:  "10.20.30.40",
		Port:  2222,
	}

	tests := []string{
		"PROD",
		"papi",
		"DePl",
		"10.30",
		"pd 2222",
	}

	for _, query := range tests {
		if !nodeMatches(node, query) {
			t.Fatalf("expected query %q to match node", query)
		}
	}
}

func TestNodeMatchesRejectsMissingKeyword(t *testing.T) {
	node := &sshw.Node{
		Name:  "Production API",
		Alias: "prod-api",
		User:  "deploy",
		Host:  "10.20.30.40",
		Port:  2222,
	}

	if nodeMatches(node, "prod staging") {
		t.Fatal("expected query with missing keyword to be rejected")
	}
}

func TestNormalizeSearchArgsAllowsEmptySearchQuery(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "search is last argument",
			args: []string{"sshw", "-search"},
			want: []string{"sshw", "-search="},
		},
		{
			name: "search precedes another flag",
			args: []string{"sshw", "-search", "-s"},
			want: []string{"sshw", "-search=", "-s"},
		},
		{
			name: "search has a keyword",
			args: []string{"sshw", "-search", "prod"},
			want: []string{"sshw", "-search", "prod"},
		},
		{
			name: "search uses equals syntax",
			args: []string{"sshw", "-search=prod"},
			want: []string{"sshw", "-search=prod"},
		},
		{
			name: "short search is last argument",
			args: []string{"sshw", "-q"},
			want: []string{"sshw", "-q="},
		},
		{
			name: "short search precedes another flag",
			args: []string{"sshw", "-q", "-s"},
			want: []string{"sshw", "-q=", "-s"},
		},
		{
			name: "short search has a keyword",
			args: []string{"sshw", "-q", "prod"},
			want: []string{"sshw", "-q", "prod"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := normalizeSearchArgs(test.args)
			if strings.Join(got, "\x00") != strings.Join(test.want, "\x00") {
				t.Fatalf("expected args %#v, got %#v", test.want, got)
			}
		})
	}
}

func TestSearchHostsReturnsAllLeafHostsForEmptyQuery(t *testing.T) {
	nodes := []*sshw.Node{
		{
			Name: "Production",
			Children: []*sshw.Node{
				{Name: "API", Host: "prod-api.example.com"},
				{Name: "Worker", Host: "prod-worker.example.com"},
			},
		},
		{Name: "Staging", Host: "staging.example.com"},
	}

	results := searchHosts(nodes, "")
	if len(results) != 3 {
		t.Fatalf("expected all 3 leaf hosts, got %d", len(results))
	}
	for _, result := range results {
		if len(result.Children) > 0 {
			t.Fatalf("expected only leaf hosts, got group %q", result.Name)
		}
	}
}

func TestRunSearchCanDeleteInitialQueryAndSearchAllHosts(t *testing.T) {
	nodes := []*sshw.Node{
		{Name: "Production", Host: "prod.example.com"},
		{Name: "Staging", Host: "staging.example.com"},
	}
	backspaces := strings.Repeat(string(promptui.KeyBackspace), len("prod"))
	stdin := io.NopCloser(strings.NewReader(backspaces + "stag\r"))
	stdout := nopWriteCloser{Writer: &bytes.Buffer{}}

	selected := runSearch(nodes, "prod", stdin, stdout)
	if selected != nodes[1] {
		t.Fatalf("expected staging host after deleting initial query, got %#v", selected)
	}
}

func TestRunSearchCanRecoverFromNoResults(t *testing.T) {
	nodes := []*sshw.Node{
		{Name: "Production", Host: "prod.example.com"},
		{Name: "Staging", Host: "staging.example.com"},
	}
	initialQuery := "missing"
	backspaces := strings.Repeat(string(promptui.KeyBackspace), len(initialQuery))
	stdin := io.NopCloser(strings.NewReader(backspaces + "stag\r"))
	stdout := nopWriteCloser{Writer: &bytes.Buffer{}}

	selected := runSearch(nodes, initialQuery, stdin, stdout)
	if selected != nodes[1] {
		t.Fatalf("expected search to remain active after no results, got %#v", selected)
	}
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error {
	return nil
}
