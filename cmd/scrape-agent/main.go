package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// codexExecCommand is the executable used to run the scraping agent prompt.
// Change this value to switch between codex binaries/wrappers.
var codexExecCommand = "codex"

// promptTemplate is intentionally separated from the command and kept at the
// top of this file for fast iteration as we tune extraction behavior.
// It must include one %s placeholder for the target URL.
var promptTemplate = "Using the playwright skill, create a list of selectors to extract blog post data from the provided URL. Use ./docs/scrape_source_guide.md as your instructions on how to inspect the site and the required output. The site is: %s"

func buildPrompt(template string, targetURL string) string {
	return fmt.Sprintf(template, targetURL)
}

func runCodex(command string, prompt string) (string, string, error) {
	cmd := exec.Command(command, "exec", prompt)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stdout.String(), stderr.String(), err
	}
	return stdout.String(), stderr.String(), nil
}

func main() {
	url := flag.String("url", "", "Target URL for scrape selector discovery")
	command := flag.String("command", codexExecCommand, "CLI executable used to run the prompt")
	template := flag.String("prompt-template", promptTemplate, "Prompt template containing one %s placeholder for URL")
	flag.Parse()

	if strings.TrimSpace(*url) == "" {
		fmt.Fprintln(os.Stderr, "missing required -url")
		os.Exit(2)
	}

	prompt := buildPrompt(*template, *url)
	stdout, stderr, err := runCodex(*command, prompt)
	if err != nil {
		if strings.TrimSpace(stderr) != "" {
			fmt.Fprint(os.Stderr, stderr)
		}
		if strings.TrimSpace(stdout) != "" {
			fmt.Fprint(os.Stdout, stdout)
		}
		os.Exit(1)
	}

	fmt.Fprint(os.Stdout, stdout)
}
