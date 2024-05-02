# emlex

A tool to extract attachments from .eml files

## Installation

- Download a prebuilt binary from [release page](https://github.com/sleepytariq/emlex/releases/latest)

  _or_
- `git clone https://github.com/sleepytariq/emlex && cd emlex && go build -ldflags="-s -w" .`

## Usage

```console
emlex [flags] email [email...]
	
    Extract attachments from multiple .eml files

Examples:
  emlex msg1.eml msg2.eml msg3.eml
  emlex *.eml
  emlex ./**/*.eml
	
Required:
  email            Path to .eml file(s)
	
Optional:
  --version        Show program version
  -h, --help       Show this message and exit
```

NOTE: *emlex* will create a new directory in the current working directory containing extracted attachments
