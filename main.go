package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jhillyerd/enmime"
	"golang.org/x/sync/semaphore"
)

const version string = "0.2.0"

func main() {
	if len(os.Args) == 1 {
		fmt.Println("Error: missing arguments, Use: -h")
		os.Exit(1)
	}

	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			ShowHelp()
			os.Exit(0)
		}
		if arg == "--version" {
			ShowVersion()
			os.Exit(0)
		}
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

	// Timestamp to make output dir unique
	ts := time.Now().Format("20060102150405")
	dir := fmt.Sprintf("%s_emlex", ts)
	err := os.Mkdir(dir, os.ModePerm)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: failed to create output directory")
		os.Exit(1)
	}

	pool := semaphore.NewWeighted(8)

	for _, email := range emails {
		for !pool.TryAcquire(1) {
		}
		go func() {
			defer pool.Release(1)
			ExtractAttachments(email, dir)
		}()
	}
}

func ExtractAttachments(path string, dir string) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to open %s\n", path)
		return
	}
	defer file.Close()

	msg, err := enmime.ReadEnvelope(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to parse %s\n", path)
		return
	}

	fmt.Printf("(%d) %s\n", len(msg.Attachments), path)

	if len(msg.Attachments) > 0 {
		var msgDateStr string
		msgDate, err := msg.Date()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to parse date from %s\n", path)
			msgDateStr = "00000000000000"
		} else {
			msgDateStr = msgDate.Format("20060102150405")
		}
		fileName := filepath.Base(path)
		attachDir := filepath.Join(dir, fmt.Sprintf("%s_%s", msgDateStr, fileName))
		os.Mkdir(attachDir, os.ModePerm)
		for _, attachment := range msg.Attachments {
			os.WriteFile(filepath.Join(attachDir, attachment.FileName), attachment.Content, 0644)
		}
	}
}

func ShowVersion() {
	fmt.Printf("emlex %s\n", version)
}

func ShowHelp() {
	fmt.Println(`emlex [flags] email [email...]
	
    Extract attachments from multiple .eml files

Examples:
  emlex msg1.eml msg2.eml msg3.eml
  emlex *.eml
  emlex ./**/*.eml
	
Required:
  email            Path to .eml file(s)
	
Optional:
  --version        Show program version
  -h, --help       Show this message and exit`)
}
