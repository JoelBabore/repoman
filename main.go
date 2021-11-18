package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/dbaggerman/cuba"
	"github.com/monochromegane/go-gitignore"
)

var FileListQueueSize = runtime.NumCPU()
var FileSummaryJobQueueSize = runtime.NumCPU()
var DirectoryWalkerJobWorkers = runtime.NumCPU()
var FileProcessJobWorkers = runtime.NumCPU()

var PathDenyList = []string{".git", ".hg", ".svn"}

//ignore files and directories matching regular expression
var Exclude = []string{}

type FileTypes struct {
	Extension string
	Count     int
}

type Outcome struct {
	Version   float32
	FileTypes []FileTypes
	// TotalDirs  int
	// TotalFiles int
}

var outcome = Outcome{
	Version: 0.1,
	// // start totalDirs on -1 to discount parent dir
	// TotalDirs:  -1,
	// TotalFiles: 0,
}

func main() {

	if len(os.Args) < 2 {
		fmt.Println()
		fmt.Println("Usage: repoman <dir>")
		fmt.Println()
		fmt.Println("Where <dir> is a source code repository as a local directory")
		fmt.Println()
		os.Exit(1)
	}

	path_to_scan := filepath.Clean(os.Args[1])

	if _, err := os.Stat(path_to_scan); os.IsNotExist(err) {
		fmt.Printf("Specified path %q does not exist", path_to_scan)
		fmt.Println()
		os.Exit(1)
	}

	// TODO spit a warning if no .git

	fileListQueue := make(chan *FileJob, FileListQueueSize)             // Files ready to be read from disk
	fileSummaryJobQueue := make(chan *FileJob, FileSummaryJobQueueSize) // Files ready to be summarised

	go func() {
		directoryWalker := NewDirectoryWalker(fileListQueue)

		err := directoryWalker.Start(path_to_scan)
		if err != nil {
			fmt.Printf("failed to walk %s: %v", path_to_scan, err)
			os.Exit(1)
		}

		directoryWalker.Run()
	}()
	go fileProcessorWorker(fileListQueue, fileSummaryJobQueue)

	fileSummarize(fileSummaryJobQueue)

	fmt.Printf("%+v\n", outcome)
}

// Run continues to run everything
func (dw *DirectoryWalker) Run() {
	dw.buffer.Finish()
	close(dw.output)
}

type FileJob struct {
	Filename  string
	Extension string
}

// NewDirectoryWalker create the new directory walker
func NewDirectoryWalker(output chan<- *FileJob) *DirectoryWalker {
	directoryWalker := &DirectoryWalker{
		output: output,
	}
	for _, exclude := range Exclude {
		regexpResult, err := regexp.Compile(exclude)
		if err == nil {
			directoryWalker.excludes = append(directoryWalker.excludes, regexpResult)
		} else {
			fmt.Println(err.Error())
		}
	}

	directoryWalker.buffer = cuba.New(directoryWalker.Walk, cuba.NewStack())
	directoryWalker.buffer.SetMaxWorkers(int32(DirectoryWalkerJobWorkers))

	return directoryWalker
}

// DirectoryWalker is responsible for actually walking directories using cuba
type DirectoryWalker struct {
	buffer   *cuba.Pool
	output   chan<- *FileJob
	excludes []*regexp.Regexp
}

// Walk walks the directory as quickly as it can
func (dw *DirectoryWalker) Walk(handle *cuba.Handle) {
	job := handle.Item().(*DirectoryJob)

	ignores := job.ignores

	dirents, err := dw.Readdir(job.path)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	for _, dirent := range dirents {
		name := dirent.Name()

		if (name == ".gitignore") || (name == ".ignore") {
			path := filepath.Join(job.path, name)

			ignore, err := gitignore.NewGitIgnore(path)
			if err != nil {
				fmt.Printf("failed to load gitignore %s: %v", job.path, err)
			} else {
				ignores = append(ignores, ignore)
			}
		}
	}

DIRENTS:
	for _, dirent := range dirents {
		name := dirent.Name()
		path := filepath.Join(job.path, name)
		isDir := dirent.IsDir()

		for _, deny := range PathDenyList {
			if strings.HasSuffix(path, deny) {
				fmt.Println("skipping", path, "due to being in denylist")
				continue DIRENTS
			}
		}

		for _, exclude := range dw.excludes {
			if exclude.Match([]byte(name)) || exclude.Match([]byte(path)) {
				fmt.Println("skipping", name, "due to match exclude")
				continue DIRENTS
			}
		}

		for _, ignore := range ignores {
			if ignore.Match(path, isDir) {
				fmt.Println("skipping", path, "due to ignore")
				continue DIRENTS
			}
		}

		if isDir {
			handle.Push(
				&DirectoryJob{
					root:    job.root,
					path:    path,
					ignores: ignores,
				},
			)
		} else {
			fileJob := newFileJob(path, name, dirent)
			if fileJob != nil {
				dw.output <- fileJob
			}
		}
	}
}

// Start actually starts directory traversal
func (dw *DirectoryWalker) Start(root string) error {
	root = filepath.Clean(root)

	fileInfo, err := os.Lstat(root)
	if err != nil {
		return err
	}

	if !fileInfo.IsDir() {
		fileJob := newFileJob(root, filepath.Base(root), fileInfo)
		if fileJob != nil {
			dw.output <- fileJob
		}

		return nil
	}

	_ = dw.buffer.Push(
		&DirectoryJob{
			root:    root,
			path:    root,
			ignores: nil,
		},
	)

	return nil
}

// DirectoryJob is a struct for dealing with directories we want to walk
type DirectoryJob struct {
	root    string
	path    string
	ignores []gitignore.IgnoreMatcher
}

func newFileJob(path, name string, fileInfo os.FileInfo) *FileJob {
	extension := getExtension(name)
	return &FileJob{
		Filename:  name,
		Extension: extension,
	}
}

// A custom version of extracting extensions for a file
// which also has a case insensitive cache in order to save
// some needless processing
func getExtension(name string) string {

	// TODO doesn't seem to like spaces in file names

	name = strings.ToLower(name)
	extension := ""

	ext := filepath.Ext(name)

	if ext == "" || strings.LastIndex(name, ".") == 0 {
		extension = name
	} else {
		// Handling multiple dots or multiple extensions only needs to delete the last extension
		// and then call filepath.Ext.
		// If there are multiple extensions, it is the value of subExt,
		// otherwise subExt is an empty string.
		subExt := filepath.Ext(strings.TrimSuffix(name, ext))
		extension = strings.TrimPrefix(subExt+ext, ".")
	}

	return extension
}

// Readdir reads a directory such that we know what files are in there
func (dw *DirectoryWalker) Readdir(path string) ([]os.FileInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return []os.FileInfo{}, fmt.Errorf("failed to open %s: %v", path, err)
	}
	defer file.Close()

	dirents, err := file.Readdir(-1)
	if err != nil {
		return []os.FileInfo{}, fmt.Errorf("failed to read %s: %v", path, err)
	}

	return dirents, nil
}

func fileSummarize(input chan *FileJob) {
	for res := range input {
		// fmt.Println(res)

		found := false
		for i := range outcome.FileTypes {
			if outcome.FileTypes[i].Extension == res.Extension {
				found = true
				outcome.FileTypes[i].Count++
				break
			}
		}
		if !found {
			outcome.FileTypes = append(outcome.FileTypes, FileTypes{
				Extension: res.Extension,
				Count:     1,
			})
		}

	}
}

// Reads and processes files from input chan in parallel, and sends results to
// output chan
func fileProcessorWorker(input chan *FileJob, output chan *FileJob) {
	// var startTime int64
	// var fileCount int64
	// var gcEnabled int64
	var wg sync.WaitGroup

	for i := 0; i < FileProcessJobWorkers; i++ {
		wg.Add(1)
		go func() {
			// reader := NewFileReader()

			for job := range input {
				// atomic.CompareAndSwapInt64(&startTime, 0, makeTimestampMilli())

				// loc := job.Location
				// if job.Symlocation != "" {
				// 	loc = job.Symlocation
				// }

				// fileStartTime := makeTimestampNano()
				// content, err := reader.ReadFile(loc, int(job.Bytes))
				// atomic.AddInt64(&fileCount, 1)

				// if atomic.LoadInt64(&gcEnabled) == 0 && atomic.LoadInt64(&fileCount) >= int64(GcFileCount) {
				// 	debug.SetGCPercent(gcPercent)
				// 	atomic.AddInt64(&gcEnabled, 1)
				// 	if Verbose {
				// 		printWarn("read file limit exceeded GC re-enabled")
				// 	}
				// }

				// if Trace {
				// 	printTrace(fmt.Sprintf("nanoseconds read into memory: %s: %d", job.Location, makeTimestampNano()-fileStartTime))
				// }

				// if err == nil {
				// job.Content = content
				// if processFile(job) {
				output <- job
				// }
				// } else {
				// 	if Verbose {
				// 		printWarn(fmt.Sprintf("error reading: %s %s", job.Location, err))
				// 	}
				// }
			}

			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(output)

		// if Debug {
		// 	printDebug(fmt.Sprintf("milliseconds reading files into memory: %d", makeTimestampMilli()-startTime))
		// }
	}()

}
