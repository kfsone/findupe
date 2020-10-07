package main

// Looks for duplicate files under a given path.
//
// see args.go for command line arguments.

import (
	"crypto/md5"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	flag "github.com/spf13/pflag"
)


// FileHash is a response to and request for file hashing.
type FileHash struct {
	// Full file and pathname of the file.
	Pathname string
	// Size is the size of the file in bytes.
	Size int64
	// Hash is where we'll the sha256 of the file.
	Hash string
}

// CollisionTable is a dictionary of file-hash -> file-list
type CollisionTable map[string][]string

// hashReqCh is the channel used to request file hashes.
var hashReqCh chan *FileHash

// hashRepCh is the channel file hashes are returned to the main thread via.
var hashRepCh chan *FileHash

// workerGroup tracks how many workers are still waiting for requests to dry up.
var workerGroup sync.WaitGroup

// Assorted global counters.
var totalFiles, underSizedFiles, hashingFiles int64


// hashData will execute a specific hashing algorithm against a file to produce the hash string.
func hashData(pathname string, hasher hash.Hash) (string, error) {
	file, err := os.Open(pathname)
	if err != nil {
		return "", err
	}

	defer file.Close()

	// Try to read the file into the hasher to obtain the hash.
	if _, err = io.Copy(hasher, file); err != nil {
		return "", err
	}

	// Produce a size+hash combination to help bucketing.
	return hex.EncodeToString(hasher.Sum(nil)), nil
}


// hashRequest will generate hash/hashes for individual files and populate the response.
func hashRequest(request *FileHash) *FileHash {
	pathname := strings.ReplaceAll(request.Pathname, "\\", "/")
	hashString, err := hashData(pathname, sha512.New())
	if err != nil {
		log.Printf("error reading %s: %s", pathname, err.Error())
		return nil
	}

	if *Thorough {
		// Extend the fingerprint with an md5 checksum.
		md5String, err := hashData(pathname, md5.New())
		if err != nil {
			log.Printf("error re-reading %s: %s", pathname, err.Error())
			return nil
		}
		hashString += "." + md5String
	}

	// Populate the request's Hash field and send it on to the reply channel.
	request.Pathname = pathname
	request.Hash = fmt.Sprintf("%016d.%s", request.Size, hashString)

	return request
}


// hashingWorker will dispatch requests for file hashes and forward the responses to the replies
// channel.
func hashingWorker(requests <-chan *FileHash, replies chan<- *FileHash) {
	// Release our contribution from the pie on exit.
	defer workerGroup.Done()

	for request := range requests {
		if reply := hashRequest(request); reply != nil {
			replies <- reply
		}
	}
}


// walkFn will receive paths from filepath.Walk and dispatch them as requests to the request
// workers via the requests channel.
func walkFn(path string, info os.FileInfo, fileErr error) (err error) {
	// Ignore directories.
	if info.IsDir() {
		return
	}

	totalFiles++

	// If there was a problem accessing the file, ignore it.
	if fileErr != nil {
		return
	}

	// Ignore zero-length files.
	if info.Size() == 0 || info.Size() < int64(*MinBytes) {
		underSizedFiles++
		return
	}

	hashingFiles++

	request := &FileHash{
		Pathname: path,
		Size:     info.Size(),
	}
	hashReqCh <- request

	return nil
}


// workers creates all of the hashing threads in the background and closes the
// reply channel once they have all exited.
func workers(requests <-chan *FileHash, replies chan<- *FileHash) {
	// When we exit scope, close the reply channel.
	defer close(replies)

	// Create workers to consume requests.
	workerGroup.Add(*Threads)
	for i := 0; i < *Threads; i++ {
		go hashingWorker(requests, replies)
	}

	// Wait for all the workers to exit.
	workerGroup.Wait()
}


// walkFiles walks the file system and closes the request channel once it
// has seen everything.
func walkFiles(requests chan<- *FileHash) {
	// When we exit, close the request channel.
	defer close(requests)

	// Start dispatching requests.
	filepath.Walk(*BasePath, walkFn)

	log.Print("Total Files:", totalFiles, ", Undersized:", underSizedFiles, ", Hashing:", hashingFiles)
}


// aggregateHashes will collect results from the reply channel and bucket filenames together
// by hash, elimiating those cases where only one file had a hash (ie it was distinct).
func aggregateHashes(replies <-chan *FileHash) CollisionTable {
	// Create dictionaries that map a file hash to a list of path names.
	// We use two dictionaries so we can filter out entries that only have
	// one file - ie nobody matched them.
	singles := make(CollisionTable)
	collisions := make(CollisionTable)

	for response := range replies {
		_, exists := collisions[response.Hash]
		if exists {
			collisions[response.Hash] = append(collisions[response.Hash], response.Pathname)
			continue
		}
		_, exists = singles[response.Hash]
		if exists {
			collisions[response.Hash] = append(singles[response.Hash], response.Pathname)
			delete(singles, response.Hash)
			continue
		}
		singles[response.Hash] = []string{response.Pathname}
	}

	collidingFiles := hashingFiles - int64(len(singles))
	duplicates := collidingFiles - int64(len(collisions))

	log.Print("Misses:", len(singles), ", Collisions:", collidingFiles, ", Hashes:", len(collisions), ", Dupes:", duplicates)

	return collisions
}


// reportCollisions will output a report of which files collided.
func reportCollisions(collisions CollisionTable) {
	for _, files := range collisions {
		for _, file := range files {
			fmt.Printf(" ")
			fmt.Printf("%q", file)
		}
		fmt.Printf("\n")
	}
}


func main() {
	var collisions CollisionTable

	flag.Parse()
	if len(flag.Args()) > 0 {
		fmt.Fprintf(os.Stderr, "\x1b[31mERROR: Unexpected argument: %s. Did you mean '--path' or is there a space in your path name?\x1b[39m\n", flag.Args()[0])
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		return
	}

	if *Threads < 1 {
		panic("--threads/-j must be >= 1")
	}
	if *MinBytes < 0 {
		*MinBytes = 0
	}

	// Create the request and reply channels.
	hashReqCh, hashRepCh = make(chan *FileHash, 65536), make(chan *FileHash, *Threads * 2)

	// Execute 'walkFiles' in the background.
	go walkFiles(hashReqCh)

	// Launch and manage the workers in the background.
	go workers(hashReqCh, hashRepCh)

	// Collect results from workers into an aggregate representation.
	collisions = aggregateHashes(hashRepCh)
	if len(collisions) == 0 {
		return
	}

	if *ListCollisions {
		reportCollisions(collisions)
	}
}
