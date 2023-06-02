package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	str "strings"
	"sync"
)

// TODO:
// - add flag and code to enable "only full match" mode
// - add benchmarks

const (
	FULL_MATCH_COLOR    = "\033[31m" // red
	PARTIAL_MATCH_COLOR = "\033[33m" // yellow
	RESET_COLOR         = "\033[0m"  // resetting the color
)

// Helper functions
func split(s, sep string) []string {
	return str.Split(s, sep)
}

func join(elems []string, sep string) string {
	return str.Join(elems, sep)
}

func trim(s string) string {
	return str.Trim(str.TrimSpace(s), "\n")
}

func addColor(elems []string, matchTypeColor string) []string {
	for i, val := range elems {
		elems[i] = matchTypeColor + val + RESET_COLOR
	}
	return elems
}

type foundResult struct {
	wasFound    bool
	coloredLine string
}

var (
	// you can hardcode a directory path in the second "" arg here if you don't want to specify the "-d" flag every time 
	dir_to_search = flag.String("d", "", "directory to use as the root for the tree to search")
	input_to_find = flag.String("w", "", "input to search for")
	waitGroup     sync.WaitGroup
)

func main() {
	// processing the passed flags
	flag.Parse()

	*dir_to_search = trim(*dir_to_search)
	*input_to_find = trim(*input_to_find)

	// exiting on empty input
	if len(str.TrimSpace(*input_to_find)) == 0 {
		fmt.Println("Error: Please specify the word to search for with a \"-w\" flag")
		os.Exit(1)
	}
	// exiting on empty input
	if len(str.TrimSpace(*dir_to_search)) == 0 {
		fmt.Println("Error: Please specify the directory to search with a \"-d\" flag")
		os.Exit(1)
	}

	// determining the directory to search
	if *dir_to_search == "cwd" || *dir_to_search == "pwd" {
		// assign current folder path to the *dir_to_search var
		wd, err := os.Getwd()
		if err != nil {
			fmt.Println("Error determining the working directory:", err)
			os.Exit(1)
		}
		*dir_to_search = filepath.Dir(wd)
	} else {
		_, err := os.Stat(*dir_to_search)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("Specified path doesn't exist:", split(err.Error(), ": ")[1], ": \"", *dir_to_search, "\"")
				os.Exit(1)
			} else {
				fmt.Println("Some error with the specified path:", err)
				os.Exit(1)
			}
		}
	}

	fmt.Print("\n\033[41m——— Found input in the following files: ———\033[0m\n\n")

	// walking the file tree
	err := filepath.WalkDir(
		*dir_to_search,
		walkFunc)
	if err != nil {
		fmt.Println("Error while walking the file tree:", err)
		os.Exit(1)
	}

	// waiting until all threads are done
	waitGroup.Wait()
}

func walkFunc(path string, d fs.DirEntry, err error) error {
	if d.Type().IsRegular() {
		waitGroup.Add(1)
		go read_file(path)
	}
	return nil
}

func read_file(path string) {
	defer waitGroup.Done()

	f, err := os.Open(path)
	if err != nil {
		fmt.Print("Error while opening a file", err, "\n\n")
		return
	} else {
		defer func() {
			if err := f.Close(); err != nil {
				fmt.Print("Error while closing the file", err, "\n\n")
			}
		}()
	}

	reader := bufio.NewReader(f)

	linecount := 0

	for {
		line, err := reader.ReadSlice('\n')
		if err != nil {
			if err.Error() == "EOF" {
				return // natural exit because of EOF
			} else if err.Error() == bufio.ErrBufferFull.Error() {
				return // this skips over the non-text files since those do not contain the '\n' delimiter and the buffer eventually gets filled up to the limit
				// There is probably a better way of doing this, but I didn't want to use file extensions as a filter parameter or read files byte-by-byte, so leaving this as is.
			}
			fmt.Print("Error while reading file", path, "\nline:", linecount, "\nerror:", err, "\n\n")
			return
		}
		linecount += 1
		if result := find_match(string(line)); result.wasFound {
			fmt.Printf("\n—————— \"%s\" on line %v:\n %s\n\n", path, linecount, trim(result.coloredLine))
		}
	}
}

func find_match(line string) foundResult {
	var result foundResult

	if found := str.Contains(line, *input_to_find); found { // if a match was found

		foundLine := split(line, " ")
		inputToFind := split(*input_to_find, " ")

		var (
			joinedFoundLine   string
			joinedInputToFind string
			foundMatch        []string
		)

		for i, word := range foundLine {
			if str.Contains(str.ToLower(word), str.ToLower(inputToFind[0])) { // "if the "word" matches with the first word of the specified input to search for"

				countWordsMatched := countMatches(foundLine[i:], inputToFind)

				joinedFoundLine = join(foundLine[i:i+countWordsMatched], "")

				joinedInputToFind = join(inputToFind, "")

				// count how many words actually matched
				foundMatch = foundLine[i : i+countWordsMatched]

				if joinedFoundLine == joinedInputToFind { // if full match found
					copy(foundMatch, addColor(foundMatch, FULL_MATCH_COLOR)) // replacing the part of that line with colored equivalent
					result = foundResult{
						wasFound:    true,
						coloredLine: join(foundLine, " "),
					}
				} else { // if partial match is found
					copy(foundMatch, addColor(foundMatch, PARTIAL_MATCH_COLOR)) // replacing the part of that line with colored equivalent
					result = foundResult{
						wasFound:    true,
						coloredLine: join(foundLine, " "),
					}
				}
			}
		}
	} else { // this would mean the input wan't found in the provided line
		result = foundResult{
			wasFound: false,
		}
	}
	return result
}

func countMatches(foundLineFromMatch []string, inputToFind []string) int {
	resultSlice := []string{}

	for i := 0; i < len(inputToFind) && i < len(foundLineFromMatch); i++ {
		if str.Contains(
			str.ToLower(foundLineFromMatch[i]),
			str.ToLower(inputToFind[i])) {
			resultSlice = append(resultSlice, inputToFind[i])
		}
	}

	return len(resultSlice)
}
