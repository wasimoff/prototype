package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	fmt.Println("This is TinyGo.")

	// print commandline arguments
	args := os.Args
	name := args[0]
	fmt.Printf("file '%s' was called with arguments: %v\n", name, args)

	// print environment variables
	fmt.Println("environment variables:")
	for _, env := range os.Environ() {
		fmt.Printf(" - %s\n", env)
	}

	// list root filesystem
	contents, err := os.ReadDir("/")
	if err != nil {
		fmt.Printf("ERR: couldn't open directory '/': %s\n", err)
	} else {
		fmt.Println("listing '/' contents:")
		for _, item := range contents {
			fmt.Printf(" - %s\n", item.Name())
		}
	}

	// try to read a specific file
	const FILENAME = "/hello.txt"
	file, err := os.Open(FILENAME)
	if err != nil {
		fmt.Printf("ERR: couldn't open file '%s': %s\n", FILENAME, err)
	} else {
		defer file.Close()
		bytes, err := io.ReadAll(file)
		if err != nil {
			fmt.Printf("ERR: couldn't read file '%s': %s\n", FILENAME, err)
		} else {
			fmt.Printf("'%s': %s\n", FILENAME, string(bytes))
		}
	}

}
