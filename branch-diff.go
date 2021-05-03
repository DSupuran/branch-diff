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
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/antchfx/xmlquery"
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

func copyFiles(files []string, directory string, forkCommit string) {
	os.RemoveAll(directory)

	if verbose {
		fmt.Printf("Deleted directory: %s\n", directory)
	}

	for _, file := range files {
		if strings.HasSuffix(file, ".profile-meta.xml") {
			content := profileDifferential(getFileFromCommit(file, forkCommit), getFileContent(file))
			writeFile(file, directory, 0600, content)
		} else {
			copyFile(file, directory, 0600)
		}

		if strings.HasSuffix(file, "-meta.xml") {
			continue
		}

		metaFile := file + "-meta.xml"
		if fileExists(metaFile) {
			copyFile(metaFile, directory, 0600)
		}
	}
}

func writeFile(file string, directory string, permissions uint32, content string) {
	destinationFile := filepath.Join(directory, file)
	path := filepath.Dir(destinationFile)

	os.MkdirAll(path, 0600)

	input := []byte(content)
	error := ioutil.WriteFile(destinationFile, input, 0600)

	if error != nil {
		log.Fatalf("file write[%s]: %s", destinationFile, error)
	}

	if verbose {
		fmt.Printf("Created file %s\n", destinationFile)
	}

}

func copyFile(file string, directory string, permissions uint32) {
	input, error := ioutil.ReadFile(file)

	if error != nil {
		log.Fatalf("file read[%s]: %s", file, error)
	}

	destinationFile := filepath.Join(directory, file)
	path := filepath.Dir(destinationFile)

	os.MkdirAll(path, 0600)

	error = ioutil.WriteFile(destinationFile, input, 0600)

	if error != nil {
		log.Fatalf("file write[%s]: %s", destinationFile, error)
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

func getFileContent(file string) string {
	output, error := ioutil.ReadFile(file)

	if error != nil {
		log.Fatalf("file read[%s]: %s", file, error)
	}

	outputString := string(output)

	return outputString	
}

func getFileFromCommit(file string, forkCommit string) string {
	commitFile := forkCommit + ":" + file

	output, error := exec.Command("git", "show", commitFile).Output()

	if error != nil {
		log.Fatalf("git show: %s", error)
	}

	return string(output)
}

// profile-diff

func profileDifferential(oldContent string, newContent string) string {
	oldChecksums := buildProfileChecksums(oldContent)
	newChecksums := buildProfileChecksums(newContent)
	newChecksumSortedKeys := sortKeys(newChecksums)

	whitelist := map[string]bool{ "custom": true, "description": true, "fullName": true, "userLicense": true }
	
	var output = `<?xml version="1.0" encoding="UTF-8"?>
<Profile xmlns="http://soap.sforce.com/2006/04/metadata">
`

	for _, checksum := range newChecksumSortedKeys {
		newNode := newChecksums[checksum]
		newNodeName := newNode.Data

		_, exists := oldChecksums[checksum]

		if !exists || whitelist[newNodeName] {
			if !whitelist[newNodeName] {

			}

			newNodeXML := newNode.OutputXML(true)
			output += newNodeXML + "\n"
		}
	}

	output += `</Profile>`

	return output
}

func sortKeys(items map[string]xmlquery.Node)[]string {
	keys := make([]string, len(items))

	i := 0
	for key := range items {
		keys[i] = key
		i++
	}

	sort.Strings(keys)

	return keys
}

func buildProfileChecksums(content string) map[string]xmlquery.Node {
	doc, err := xmlquery.Parse(strings.NewReader(content))
	if err != nil {
		panic(err)
	}

	nodes := xmlquery.Find(doc, "//Profile/*")
	checksums := make(map[string]xmlquery.Node)

	for _, node := range nodes {
		nodeXML := node.OutputXML(true)
		nodeName := node.Data
		sha256Hex := sha256Hex(nodeXML)

		key := nodeName + "|" + sha256Hex
		checksums[key] = *node
	}

	return checksums

}

func sha256Hex(content string) string {
	contentBytes := []byte(content)
	sha256Bytes := sha256.Sum256(contentBytes)
	sha256Hex := hex.EncodeToString(sha256Bytes[:])

	return sha256Hex
}
