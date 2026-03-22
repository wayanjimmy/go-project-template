package main

import (
	"flag"
	"fmt"
	"os"
)

// Cli tools for managing elastic index
func main() {
	var action string
	var indexName string
	var aliasName string

	flag.StringVar(&action, "action", "", "create-index, delete-index or put-alias")
	flag.StringVar(&indexName, "index", "", "index name")
	flag.StringVar(&aliasName, "alias", "blacklist", "alias name")

	flag.Parse()

	switch action {
	case "create-index":
		fmt.Printf("Your action was %s", action)
	case "delete-index":
		fmt.Printf("Your action was %s", action)
	default:
		fmt.Fprintln(os.Stderr, "You must specify the action")
		os.Exit(1)
	}
}
