Find Duplicates
===============

Go code to find duplicate files by using size+sha512, with option to also use md5 sums to deepen
confidence in matches.


# Installation

findupe is written in go-lang and is provided as source, so you'll need to have Google's
"go" language tools available.


## Golang Installation

_Windows_
```pwsh
	winget install golang
```

_Debian_
```bash
	sudo apt install golang
	# or
	snap install --classic go
```

_Redhat/CentOS_
```bash
	sudo yum install golang
	# or
	snap install --classic go
```

_MacOS_
``` zsh
	brew install golang
```

## Installing Findupe

	go get -u github.com/kfsone/findupe

You can then optionally either install it, which requires ~/go/bin to be in your path:

	go install github.com/kfsone/findupe

Or you can run it from the command line with go run:

	go run github.com/kfsone/findupe


## Usage

    -L, --list-collisions   List files for which matches were found.
    -b, --min-bytes int     Minimum size (bytes) for file to consider. (default 256)
    -p, --path string       Directory to recurse over. (default ".")
    -T, --thorough          Append SHA sums with MD5 sums.
    -j, --threads int       Number of concurrent workers. (default 9)


# Examples

Search for duplicates under some path. with only a summary.

	findupe --path some/sub/dir


Search '/tmp' for files over 1k in size, use both SHA 512 and MD5 sums to maximize confidence
that duplicates are exact matches, use 32 concurrent workers, and list the actual files.

	findupe -b 1024 --list-collisions -T -p /tmp

