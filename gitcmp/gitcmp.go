/*
Binary gitcmp compares two remote Git repositories.

Sample use:
	$ go get github.com/dsymonds/gitutil/gitcmp
	$ gitcmp https://github.com/bradfitz/camlistore.git https://camlistore.googlesource.com/camlistore | head -5
	refs/heads/master differs: b38077ff0fecee907d8b1338f904ccc97b9d0f2a vs. 90d1df956f50431fdd41979b37c164be1daf2488
	HEAD differs: b38077ff0fecee907d8b1338f904ccc97b9d0f2a vs. 90d1df956f50431fdd41979b37c164be1daf2488
	refs/tags/0.6 differs: 289be262e1206cd71b70ec3c75024f4efa170c69 vs. 84ea8092ef434dbd7c077b9254354d429ded5e51
	Only in https://github.com/bradfitz/camlistore.git: refs/pull/9/head
	Only in https://github.com/bradfitz/camlistore.git: refs/pull/9/merge
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/dsymonds/gitutil"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s <repo1> <repo2>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(2)
	}

	repo1, repo2 := flag.Arg(0), flag.Arg(1)

	var refs1, refs2 map[string]string
	fetch := func(u string) map[string]string {
		refs, err := gitutil.RemoteRefs(nil, u)
		if err != nil {
			log.Fatalf("Fetching %s: %v", u, err)
		}
		return refs
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		refs1 = fetch(repo1)
	}()
	go func() {
		defer wg.Done()
		refs2 = fetch(repo2)
	}()
	wg.Wait()

	// Diff
	ok := true
	for ref, hash1 := range refs1 {
		hash2, ok := refs2[ref]
		if !ok {
			continue
		}
		if hash1 != hash2 {
			fmt.Printf("%s differs: %s vs. %s\n", ref, hash1, hash2)
			ok = false
		}
		delete(refs1, ref)
		delete(refs2, ref)
	}
	only := func(u string, refs map[string]string) {
		for ref := range refs {
			fmt.Printf("Only in %s: %s\n", u, ref)
		}
	}
	only(repo1, refs1)
	only(repo2, refs2)
	if !ok {
		os.Exit(1)
	}
}
