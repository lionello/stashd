// Copyright Lionello Lunesu. Placed in the public Domain.
// https://github.com/lionello/stashd

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		usage(os.Args[0])
		os.Exit(129)
	}

	// Check if PAGER is set
	pager := os.Getenv("PAGER")
	if pager != "" {
		cmd := exec.Command("/bin/sh", append([]string{"-c"}, pager)...)
		pagerStdin, err := cmd.StdinPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating pipe: %v\n", err)
			os.Exit(1)
		}

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting pager: %v\n", err)
			os.Exit(1)
		}

		runStashd(os.Args, pagerStdin)
		pagerStdin.Close()
		cmd.Wait()
	} else {
		runStashd(os.Args, os.Stdout)
	}
}

func runStashd(args []string, output io.Writer) {
	// FIXME: support proper command line arguments
	filenames := args[1:]
	var extraArgs []string

	if detectTerminal() {
		extraArgs = append(extraArgs, "--color=always")
	}

	for _, arg := range filenames {
		if arg == "--" {
			break
		}
		if arg == "-h" || arg == "--help" {
			usage(args[0])
			os.Exit(129)
		}
		if len(arg) >= 2 && arg[0] == '-' {
			extraArgs = append(extraArgs, arg)
		}
	}

	gitArgs := append([]string{"stash", "list", "-p"}, extraArgs...)
	cmd := exec.Command("git", gitArgs...)
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating stdout pipe: %v\n", err)
		os.Exit(1)
	}

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting git: %v\n", err)
		os.Exit(1)
	}

	const stashPrefix = "stash@{"
	const diffPrefix = "diff --git a/"

	scanner := bufio.NewScanner(stdout)
	var lastStash string
	dump := false

	for scanner.Scan() {
		line := scanner.Text()

		if len(line) > 10 {
			start := 0
			if strings.HasPrefix(line, "\x1b") {
				start = 4
			}
			end := start + len(diffPrefix)

			if len(line) > end && line[start:end] == diffPrefix {
				// Encountered a new diff; stop dumping and check filenames
				dump = false
				if anyMatch(filenames, line[end:]) {
					// Found a match; print stash header if not yet printed
					if lastStash != "" {
						fmt.Fprintln(output, lastStash)
						fmt.Fprintln(output)
						lastStash = ""
					}
					dump = true
				}
			} else if strings.HasPrefix(line, stashPrefix) {
				// Encountered a new stash; stop dumping but save header in case a file matches
				dump = false
				lastStash = line
			}
		}

		if dump {
			fmt.Fprintln(output, line)
		}
	}

	cmd.Wait()
}

func anyMatch(names []string, line string) bool {
	for _, name := range names {
		if strings.Contains(line, name) {
			return true
		}
	}
	return false
}

func usage(arg0 string) {
	fmt.Printf("Usage: %s [<diff options>] [--] filename...\n", filepath.Base(arg0))
}

func detectTerminal() bool {
	term := os.Getenv("TERM")
	stat, _ := os.Stdout.Stat()
	return (stat.Mode()&os.ModeCharDevice != 0) && term != "" && term != "dumb"
}
