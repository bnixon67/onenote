package main

import (
	"encoding/json"
	"fmt"
	"github.com/bnixon67/onenote"
	//	"golang.org/x/net/html"
	"io/ioutil"
	"net/url"
	"os"
	//	"strings"
	"log"
)

// readToken reads the access token from the given filename
func readToken(filename string) []byte {
	// open file
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// read file
	token, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	return (token)
}

// writeContent writes out the content
func writeContent(fileName string, content string) {
	// create file
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	// write content
	_, err = file.WriteString(content)
	if err != nil {
		log.Fatal(err)
	}

	return
}

// showResponse pretty prints the response using MarshalIndent
func showResponse(v interface{}) {
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		fmt.Println("error: err")
	}
	fmt.Printf("==========\n%s\n===========\n", b)
}

func main() {
	// read the access token, set via authorize.go
	token := readToken("token.txt")

	var query url.Values

	// ----- List Notebooks
	query = url.Values{}
	query.Set("$count", "true")
	//query.Set("$top", "1")
	//query.Set("$filter", "startswith(displayName, 'U')")
	notebooksResponse := onenote.ListNotebooks(token, query)

	fmt.Printf("total notebooks = %d\n", notebooksResponse.ODataCount)
	fmt.Printf("response notebooks = %d\n\n", len(notebooksResponse.Value))
	//showResponse(notebooksResponse)

	for n, notebook := range notebooksResponse.Value {
		fmt.Printf("notebook[%d]\t%s\n", n, notebook.DisplayName)
	}
	fmt.Println()
}
