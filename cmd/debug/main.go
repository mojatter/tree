package main

import (
	"fmt"
	"os"
	"regexp"
)

var tokenRegexp = regexp.MustCompile(`"([^"]*)"|(and|or|==|<=|>=|!=|~=|\.\.|[\.\[\]\(\)\|<>:]|([a-z]+)\(([^\)]*)\))|(\w+)`)

func main() {
	expr := os.Args[1]
	ms := tokenRegexp.FindAllStringSubmatch(expr, -1)
	for _, m := range ms {
		fmt.Printf("%#v\n", m)
	}
}
