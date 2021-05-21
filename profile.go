package branchdiff

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"github.com/antchfx/xmlquery"
)

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