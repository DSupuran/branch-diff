package branchdiff

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func pathSuffix(parts []string, count int) string {
	var outputPathname string
	
	for i := 0; i <= count; i++ {
		outputPathname = filepath.Join(outputPathname, parts[i])
	}
	
	return outputPathname
}

func copyDir(pathname string, suffix string) {
	fileinfo, error := os.Stat(pathname)
	if error != nil {
		fmt.Printf("WARNING [os.Stat(%s): %s\n", pathname, error)
		return
	}
	if !fileinfo.IsDir() {
		fmt.Errorf("source is not a directory")
		return
	}

	destinationPathname := filepath.Join(suffix, pathname)

	if isDirectory(destinationPathname) {
		if verbose {
			fmt.Printf("directory already exists[skipping]: %s\n", destinationPathname)
		}
		
		return
	}

	error = os.MkdirAll(destinationPathname, fileinfo.Mode())
	if error != nil {
		return
	}

	nodes, error := ioutil.ReadDir(pathname)
	if error != nil {
		return
	}

	for _, node := range nodes {
		srcPath := filepath.Join(pathname, node.Name())

		if node.IsDir() {
			copyDir(srcPath, suffix)
		} else {
			// Skip symlinks.
			if node.Mode()&os.ModeSymlink != 0 {
				continue
			}

			copyFile(srcPath, suffix, 0600)
		}
	}

	return
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

func isDirectory(pathname string) bool {
	fileinfo, error := os.Stat(pathname)
	
	if error != nil {
		return false
	}
	
	return fileinfo.IsDir()
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
