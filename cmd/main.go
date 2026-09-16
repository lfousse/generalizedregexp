// A command line tool accepting a generalized regexp as argument.
package main

import (
	"fmt"
	"os"

	"github.com/lfousse/generalizedregexp/filter"
)

func main() {
	r, err := filter.Parse(os.Args[1:])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := filter.Lines(r, os.Stdin, os.Stdout); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
