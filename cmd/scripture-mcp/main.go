// scripture-mcp is the verse-driven binary. One executable, five
// subcommands:
//
//	scripture-mcp serve                 — stdio MCP server for Claude/Codex
//	scripture-mcp lookup "<ref>"        — print a verse (json|text)
//	scripture-mcp lookup-from-prompt    — hook integration: stdin → JSON
//	scripture-mcp recap [flags]         — Mode B terminal-only recap
//	scripture-mcp init --target=...     — splice config snippets into agents
//
// With no subcommand, prints version + pack summary.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/MiaoDX/verse-driven/internal/cli"
	"github.com/MiaoDX/verse-driven/internal/mcp"
	"github.com/MiaoDX/verse-driven/internal/packs"
)

const Version = "v0.1.0"

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("scripture-mcp %s\n", Version)
		fmt.Printf("packs loaded: %d  total verses: %d\n",
			len(packs.All().Names()), packs.All().TotalVerses())
		fmt.Println("usage: scripture-mcp {serve|lookup|lookup-from-prompt|recap|init} [flags]")
		return
	}
	streams := cli.Streams{In: os.Stdin, Out: os.Stdout, Err: os.Stderr}
	sub := os.Args[1]
	args := os.Args[2:]
	switch sub {
	case "-h", "--help", "help":
		printHelp()
		return
	case "-v", "--version", "version":
		fmt.Println(Version)
		return
	case "serve":
		os.Exit(runServe(args))
	default:
		os.Exit(cli.Run(sub, args, streams))
	}
}

func printHelp() {
	fmt.Println("scripture-mcp", Version)
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  scripture-mcp serve                              start stdio MCP server")
	fmt.Println("  scripture-mcp lookup \"<ref>\" [--format=json|text] resolve and print a verse")
	fmt.Println("  scripture-mcp lookup-from-prompt                 hook integration (stdin → JSON)")
	fmt.Println("  scripture-mcp recap [--tradition=<t>] [--terminal] [--first-letter] [--seed=<n>]")
	fmt.Println("  scripture-mcp init --target={claude-code|codex} [--recap=on|off] [--uninstall]")
}

func runServe(args []string) int {
	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, "error: serve takes no arguments")
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	srv := mcp.New(packs.All())
	if err := srv.Serve(ctx, os.Stdin, os.Stdout); err != nil && err != context.Canceled {
		fmt.Fprintln(os.Stderr, "scripture-mcp serve:", err)
		return 1
	}
	return 0
}
