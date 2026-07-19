// Command multi-section-patch reads and edits exact named sections across multiple text files.
package main

import (
	"os"

	"github.com/rudra2112/multi-section-patch/internal/multisectionpatch"
)

func main() {
	os.Exit(multisectionpatch.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
