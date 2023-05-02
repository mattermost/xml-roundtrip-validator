package main

import (
	"flag"
	"fmt"
	"os"

	validator "github.com/mattermost/xml-roundtrip-validator"
)

func main() {
	all := flag.Bool("all", false, "Validate the entire document instead of bailing out on the first error")
	flag.Parse()

	file := flag.Arg(0)

	if file == "" {
		fmt.Fprintln(os.Stderr, "Specify a filename")
		os.Exit(1)
	}

	f, err := os.Open(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if *all {
		errs := validator.ValidateAll(f)
		if len(errs) == 0 {
			fmt.Println("Document validated without errors")
			os.Exit(0)
		}
		for _, err := range errs {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		os.Exit(1)
	}
	err = validator.Validate(f)
	if err == nil {
		fmt.Println("Document validated without errors")
		os.Exit(0)
	}
	fmt.Fprintf(os.Stderr, "%v\n", err)
	os.Exit(1)
}
