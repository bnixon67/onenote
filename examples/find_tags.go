package main

import (
	"encoding/json"
	"fmt"
	"github.com/bnixon67/onenote"
	"golang.org/x/net/html"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"runtime/pprof"
	"runtime"
	"flag"
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

// find_tag find the given tag in the source using HTML Tokenizer
//   returns a string array of the tag values
func find_tag(source io.Reader, tag string) (vals []string) {

	z := html.NewTokenizer(source)

	saveText := false
	for {
		tokenType := z.Next()

		switch tokenType {

		case html.ErrorToken:
			return

		case html.TextToken:
			if saveText {
				vals = append(vals, string(z.Text()))
			}

		case html.StartTagToken:
			_, hasAttr := z.TagName()
			if hasAttr {
				key, val, _ := z.TagAttr()
				if string(key) == "data-tag" {
					vals := strings.Split(string(val), ",")
					for _, v := range vals {
						if v == tag {
							saveText = true
						}
					}
				}
			}
		case html.EndTagToken:
			saveText = false
		}
	}
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

func main() {
//	runtime.GOMAXPROCS(1)

	   flag.Parse()
	   if *cpuprofile != "" {
	       f, err := os.Create(*cpuprofile)
	       if err != nil {
	           log.Fatal("could not create CPU profile: ", err)
	       }
	       if err := pprof.StartCPUProfile(f); err != nil {
	           log.Fatal("could not start CPU profile: ", err)
	       }
	       defer pprof.StopCPUProfile()
	   }

	// read the access token, set via authorize.go
	token := readToken("token.txt")

	var query url.Values
	var nextLink *url.URL

	// WaitGroup to fetch multiple pages
	var wg sync.WaitGroup

	// list of page may be returned by multiple queries
	// @odata.nextLink has the link to next set of pages
	for {
		// ----- List Pages

		// first query (no nextLink)
		if nextLink == nil {
			query = url.Values{}

			// total number of pages
			query.Set("$count", "true")

			// sort by page title
			//			query.Set("$orderby", "parentSection/displayName,title")
			query.Set("$orderby", "title")

			// exand parentNotebook to get displayName
			query.Set("$expand", "parentNotebook,parentSection")

			// filter on just one Notebook
			query.Set("$filter",
				"parentNotebook/displayName eq 'UMB Notes'")
		} else {
			// set query value based on nextLink
			query = nextLink.Query()
		}

		// get pages
		pagesResponse := onenote.ListPages(token, query)

		// get nextLink (if any)
		nextLink, _ = url.Parse(pagesResponse.ODataNextLink)

		// loop thru each page
		for _, page := range pagesResponse.Value {

			// increase WaitGroup counter
			wg.Add(1)

			// run goroutine to get page content and find tags
			go func(page onenote.Page) {
				// ensure we decrease WaitGroup
				defer wg.Done()

				// ----- Get Page Content
				content := onenote.GetPageContent(token, page.Id, nil)

				// find to-do tags in the page content
				v := find_tag(strings.NewReader(content), "to-do")

				// at least one to-do tag found
				if len(v) > 0 {
					fmt.Printf("----- %3d %s/%s/%s\n",
						len(v),
						page.ParentNotebook.DisplayName,
						page.ParentSection.DisplayName,
						page.Title)
						for n, v := range v {
							fmt.Printf("%3d\t%v\n", n, v)
						}
						fmt.Println()
				}
			}(page)

			// ----- Write Page Content
			//writeContent(page.Id+".html", content)
		}

		// Wait for all page requests complete
		wg.Wait()

		// nextLink is empty, so exit loop
		if pagesResponse.ODataNextLink == "" {
			break
		}

	}

	   if *memprofile != "" {
	        f, err := os.Create(*memprofile)
	        if err != nil {
	            log.Fatal("could not create memory profile: ", err)
	        }
	        runtime.GC() // get up-to-date statistics
	        if err := pprof.WriteHeapProfile(f); err != nil {
	            log.Fatal("could not write memory profile: ", err)
	        }
	        f.Close()
	    }
}
