package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jhillyerd/enmime"
)

const version string = "0.2.6"

type Attachment struct {
	Name string
	Data []byte
}

func main() {
	if len(os.Args) == 1 {
		fmt.Println("missing arguments, Use: -h")
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
			fmt.Fprintf(os.Stderr, "failed to resolve %s\n", input)
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
		fmt.Fprintln(os.Stderr, "no valid files were passed")
		os.Exit(1)
	}

	// Timestamp to make output dir unique
	ts := time.Now().Format("20060102150405")
	dir := fmt.Sprintf("%s_emlex", ts)
	err := os.Mkdir(dir, os.ModePerm)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to create output directory")
		os.Exit(1)
	}

	wg := new(sync.WaitGroup)
	sem := make(chan bool, 8)

	for _, email := range emails {
		sem <- true
		wg.Add(1)
		go func() {
			defer func() { wg.Done(); <-sem }()
			attachments, err := ExtractAttachments(email)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			attachDir := filepath.Join(dir, strings.ReplaceAll(filepath.Clean(email), string(os.PathSeparator), " → "))
			err = SaveAttachments(attachDir, attachments)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
		}()
	}
	wg.Wait()
}

func ExtractAttachments(path string) (*[]Attachment, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s", path)
	}
	defer file.Close()

	msg, err := enmime.ReadEnvelope(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s", path)
	}

	if len(msg.Attachments) == 0 {
		return nil, fmt.Errorf("%s does not contain attachments", path)
	}

	var attachments []Attachment

	for _, attachment := range msg.Attachments {
		attachments = append(attachments, Attachment{
			Name: attachment.FileName,
			Data: attachment.Content,
		})
	}
	return &attachments, nil
}

func SaveAttachments(dir string, attachments *[]Attachment) error {
	err := os.Mkdir(dir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create %s", dir)
	}
	for _, attachment := range *attachments {
		err := os.WriteFile(filepath.Join(dir, attachment.Name), attachment.Data, 0644)
		if err != nil {
			return fmt.Errorf("failed to save %s in %s", attachment.Name, dir)
		}
	}
	return nil
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
