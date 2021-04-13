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

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
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

func main() {
	initializeParameters()

	forkCommit := forkPoint()

	files := changeList(forkCommit, currentCommit)

	for _, file := range files {
		if verbose {
			fmt.Printf("File change found: %s\n", file)
		}
	}

	copyFiles(files, outputDirectory)

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
		log.Fatal("git merge-base: %s", error)
	}

	hash := strings.ReplaceAll(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n", "")

	return hash
}

func changeList(startCommit string, endCommit string) []string {
	output, error := exec.Command("git", "diff", "--name-only", startCommit, endCommit).Output()

	if error != nil {
		log.Fatal("git diff: %s", error)
	}

	delimiterIgnoreEmptyItems := func(c rune) bool {
		return c == '\n'
	}

	lines := strings.FieldsFunc(strings.ReplaceAll(string(output), "\r\n", "\n"), delimiterIgnoreEmptyItems)

	return lines
}

func copyFiles(files []string, directory string) {
	os.RemoveAll(directory)

	if verbose {
		fmt.Printf("Deleted directory: %s\n", directory)
	}

	for _, file := range files {
		copyFile(file, directory, 0600)

		metaFile := file + "-meta.xml"
		if fileExists(metaFile) {
			copyFile(metaFile, directory, 0600)
		}
	}
}

func copyFile(file string, directory string, permissions uint32) {
	input, error := ioutil.ReadFile(file)

	if error != nil {
		log.Fatal("file read[%s]: %s", file, error)
	}

	destinationFile := filepath.Join(directory, file)
	path := filepath.Dir(destinationFile)

	os.MkdirAll(path, 0600)

	error = ioutil.WriteFile(destinationFile, input, 0600)

	if error != nil {
		log.Fatal("file write[%s]: %s", destinationFile, error)
	}

	if verbose {
		fmt.Printf("Created file %s\n", destinationFile)
	}

}

func fileExists(file string) bool {
	info, error := os.Stat(file)

	if os.IsNotExist(error) {
		return false
	}

	return !info.IsDir()
}
