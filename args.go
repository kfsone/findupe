package main

// Command line arguments.

import (
	flag "github.com/spf13/pflag"
)

// BasePath is the top-level of the crawl.
var BasePath = flag.StringP("path", "p", ".", "Directory to recurse over.")

// MinBytes specifies the minimum size a file must be to be compared.
var MinBytes = flag.IntP("min-bytes", "b", 256, "Minimum size (bytes) for file to consider.")

// Jobs (threads) is how many workers to run concurrently.
var Threads = flag.IntP("threads", "j", 9, "Number of concurrent workers.")

// Thorough will do an md5 on files after the sha512.
var Thorough = flag.BoolP("thorough", "T", false, "Append SHA sums with MD5 sums.")

// Present a listing of all the collisions.
var ListCollisions = flag.BoolP("list-collisions", "L", false, "List files for which matches were found.")