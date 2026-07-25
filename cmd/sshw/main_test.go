package main

import (
	"testing"

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
