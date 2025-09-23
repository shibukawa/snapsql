package testrunner

import (
	"os"
	"path/filepath"
	"strings"
)

// walkAndProcessFiles walks a path (file or directory) and invokes onFile for each file.
// It skips common VCS/vendor directories and supports skipping the root directory filtering.
func walkAndProcessFiles(root string, allowSkipRoot bool, onFile func(p string, info os.FileInfo)) error {
	info, err := os.Stat(root)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				if !allowSkipRoot && p == root {
					return nil
				}

				name := info.Name()
				if name == "vendor" || name == ".git" || name == "node_modules" || strings.HasPrefix(name, ".") {
					if p == root {
						return nil
					}

					return filepath.SkipDir
				}

				return nil
			}

			onFile(p, info)

			return nil
		})
	}

	onFile(root, info)

	return nil
}
