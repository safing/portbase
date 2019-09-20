// +build !windows

package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func ExampleDirStructure() {
	// output:
	// / [755]
	// /repo [777]
	// /repo/b [755]
	// /repo/b/c [750]
	// /repo/b/d [755]
	// /repo/b/d/e [755]
	// /repo/b/d/f [755]
	// /secret [700]

	basePath, err := ioutil.TempDir("", "")
	if err != nil {
		fmt.Println(err)
		return
	}

	ds := NewDirStructure(basePath, 0755)
	secret := ds.ChildDir("secret", 0700)
	repo := ds.ChildDir("repo", 0777)
	_ = repo.ChildDir("a", 0700)
	b := repo.ChildDir("b", 0755)
	c := b.ChildDir("c", 0750)

	err = ds.Ensure()
	if err != nil {
		fmt.Println(err)
	}

	err = c.Ensure()
	if err != nil {
		fmt.Println(err)
	}

	err = secret.Ensure()
	if err != nil {
		fmt.Println(err)
	}

	err = b.EnsureRelDir("d", "e")
	if err != nil {
		fmt.Println(err)
	}

	err = b.EnsureRelPath("d/f")
	if err != nil {
		fmt.Println(err)
	}

	_ = filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err == nil {
			dir := strings.TrimPrefix(path, basePath)
			if dir == "" {
				dir = "/"
			}
			fmt.Printf("%s [%o]\n", dir, info.Mode().Perm())
		}
		return nil
	})
}
