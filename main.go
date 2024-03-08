package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jhillyerd/enmime"
)

const version string = "0.1.0"

func main() {
	if len(os.Args) == 1 {
		fmt.Println("Error: missing arguments, Use: -h")
		os.Exit(1)
	}

	if slices.Contains(os.Args[1:], "-h") || slices.Contains(os.Args[1:], "--help") {
		ShowHelp()
		os.Exit(0)
	}

	if slices.Contains(os.Args[1:], "-v") || slices.Contains(os.Args[1:], "--version") {
		ShowVersion()
		os.Exit(0)
	}

	// resolve all input files
	var emails []string

	for _, input := range os.Args[1:] {
		files, err := filepath.Glob(input)
		if err != nil || len(files) == 0 {
			fmt.Fprintf(os.Stderr, "Error: failed to resolve %s\n", input)
			continue
		}
		for _, file := range files {
			stat, _ := os.Stat(file)
			if stat.IsDir() {
				continue
			}
			emails = append(emails, file)
		}
	}

	if len(emails) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no valid files were passed")
		os.Exit(1)
	}

	ts := time.Now().Format("20060102150405")

	for _, email := range emails {
		file, err := os.Open(email)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to open %s\n", email)
			continue
		}
		defer file.Close()

		msg, err := enmime.ReadEnvelope(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to parse %s\n", email)
			continue
		}

		fmt.Printf("(%d) %s\n", len(msg.Attachments), email)

		for _, attachment := range msg.Attachments {
			dir := filepath.Join(fmt.Sprintf("%s_emlex", ts), strings.Split(filepath.Base(email), ".eml")[0])
			os.MkdirAll(dir, os.ModePerm)
			os.WriteFile(filepath.Join(dir, attachment.FileName), attachment.Content, 0644)
		}
	}
}

func ShowVersion() {
	fmt.Printf("emlex %s\n", version)
}

func ShowHelp() {
	fmt.Println(`emlex [-h] [-v] email [email...]
	
    Extract attachments from multiple .eml files

Examples:
  emlex msg1.eml msg2.eml msg3.eml
  emlex *.eml
	
Required:
  email            Path to .eml file(s)
	
Optional:
  --version        Show program version
  -h, --help       Show this message and exit`)
}
