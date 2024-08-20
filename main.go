package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jhillyerd/enmime"
)

const version string = "0.3.1"

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
			to, subject, attachments, err := ParseEmail(email)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			for _, address := range to {
				attachDir := filepath.Join(dir, address, RemoveIllegalChars(subject))
				err = SaveAttachments(attachDir, attachments)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					return
				}
				err = CopyFileToDst(email, attachDir)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func ParseEmail(path string) ([]string, string, *[]Attachment, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to open %s", path)
	}
	defer file.Close()

	msg, err := enmime.ReadEnvelope(file)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to parse %s", path)
	}

	if len(msg.Attachments) == 0 {
		return nil, "", nil, fmt.Errorf("%s does not contain attachments", path)
	}

	var attachments []Attachment

	for _, attachment := range msg.Attachments {
		attachments = append(attachments, Attachment{
			Name: attachment.FileName,
			Data: attachment.Content,
		})
	}

	addressPattern := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	to := addressPattern.FindAllString(msg.GetHeader("To"), -1)
	slices.Sort(to)
	to = slices.Compact(to)

	subject := msg.GetHeader("Subject")
	if subject == "" {
		subject = uuid.NewString()
	}

	return to, subject, &attachments, nil
}

func SaveAttachments(dir string, attachments *[]Attachment) error {
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create %s", dir)
	}
	for _, attachment := range *attachments {

		// in case of empty filename
		if attachment.Name == "" {
			attachment.Name = uuid.NewString()
		}

		err := os.WriteFile(filepath.Join(dir, attachment.Name), attachment.Data, 0644)
		if err != nil {
			return fmt.Errorf("failed to save %s in %s", attachment.Name, dir)
		}
	}
	return nil
}

func RemoveIllegalChars(s string) string {
	pattern := regexp.MustCompile(`[/\\?%*:|"<>]`)
	s = pattern.ReplaceAllString(s, "_")
	return s
}

func CopyFileToDst(src string, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to copy source email %s", src)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(filepath.Join(dst, "original.eml"))
	if err != nil {
		return fmt.Errorf("failed to copy source email %s", src)
	}
	defer dstFile.Close()

	io.Copy(dstFile, srcFile)
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
