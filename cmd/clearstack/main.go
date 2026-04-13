// Command clearstack is a safe, cross-platform cleaner for developer disk artifacts.
//
// See https://github.com/guilhermejansen/clearstack for docs.
package main

import (
	"fmt"
	"os"

	"github.com/guilhermejansen/clearstack/internal/version"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println(version.Full())
		return
	}
	fmt.Printf("clearstack %s — bootstrap phase, CLI coming in Sprint 2.\n", version.Full())
	fmt.Println("See https://github.com/guilhermejansen/clearstack")
}
