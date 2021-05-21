/*
	Copyright David Supuran, 2021

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package branchdiff

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var outputDirectory string
var currentCommit string
var parentBranch string
var verbose bool

var allOrNothingComponentTypes = map[string]bool {
	"aura": true,
	"experiences": true,
	"lwc": true,
}

func main() {
	initializeParameters()

	forkCommit := forkPoint()

	files := changeList(forkCommit, currentCommit)

	for _, file := range files {
		if verbose {
			fmt.Printf("File change found: %s\n", file)
		}
	}

	copyFiles(files, outputDirectory, forkCommit)

	return
}

func initializeParameters() {
	flag.StringVar(&outputDirectory, "directory", "deploy", "Output diretory to copy modified changes into")
	flag.StringVar(&currentCommit, "current", "HEAD", "Current commit/branch to compare against")
	flag.StringVar(&parentBranch, "parent", "develop", "Parent commit/branch to compare against")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")
	flag.Parse()
}

func forkPoint() string {
	output, error := exec.Command("git", "merge-base", parentBranch, currentCommit).Output()

	if error != nil {
		log.Fatalf("git merge-base: %s", error)
	}

	hash := strings.ReplaceAll(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n", "")

	return hash
}

func changeList(startCommit string, endCommit string) []string {
	output, error := exec.Command("git", "diff", "--name-only", startCommit, endCommit).Output()

	if error != nil {
		log.Fatalf("git diff: %s", error)
	}

	delimiterIgnoreEmptyItems := func(c rune) bool {
		return c == '\n'
	}

	lines := strings.FieldsFunc(strings.ReplaceAll(string(output), "\r\n", "\n"), delimiterIgnoreEmptyItems)

	return lines
}

func copyFiles(files []string, suffix string, forkCommit string) {
	os.RemoveAll(suffix)

	if verbose {
		fmt.Printf("Deleted directory: %s\n", suffix)
	}

	for _, file := range files {
	
		pathname := filterComponentFilename(file)
	
		if strings.HasSuffix(file, ".profile-meta.xml") {
			content := profileDifferential(getFileFromCommit(file, forkCommit), getFileContent(file))
			writeFile(file, suffix, 0600, content)
		} else {
			if isDirectory(pathname) {
				copyDir(pathname, suffix)
			} else {
				copyFile(file, suffix, 0600)
			}
		}

		if strings.HasSuffix(file, "-meta.xml") {
			continue
		}

		metaFile := file + "-meta.xml"
		if fileExists(metaFile) {
			copyFile(metaFile, suffix, 0600)
		}
	}
}

func filterComponentFilename(pathname string) string {
	outputPathname := pathname
	
	fmt.Printf("Checking path = %s\n", filepath.Dir(pathname))
	parts := strings.Split(filepath.Dir(pathname), string(os.PathSeparator))
	
	for i := 0; i < len(parts) - 1; i++ {
		_, exists := allOrNothingComponentTypes[parts[i]]
		
		if exists {
			fmt.Printf("i=%d, part=%s\n", i, parts[i])
			outputPathname = pathSuffix(parts, i+1)
		}
	}
	
	return outputPathname
}