package main

import (
	"fmt"
	"golang.org/x/net/html"
	"io"
	"os"
	"strings"
)

func find_tag(source io.Reader, tag string) {

	z := html.NewTokenizer(source)

	todo := false
	for {
		tokenType := z.Next()

		switch tokenType {

		case html.ErrorToken:
			return

		case html.TextToken:
			if todo {
				fmt.Println(string(z.Text()))
				fmt.Println()
			}

		case html.StartTagToken:
			_, hasAttr := z.TagName()
			if hasAttr {
				key, val, _ := z.TagAttr()
				if string(key) == "data-tag" {
					vals := strings.Split(string(val), ",")
					for _, v := range vals {
						if v == tag {
							todo = true
						}
					}
				}
			}
		case html.EndTagToken:
			todo = false
		}
	}
}

func main() {
	file, _ := os.Open(os.Args[1])

	find_tag(file, "to-do")
}
