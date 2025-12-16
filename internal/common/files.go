package common

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func CollectJPEGs(root string, recursive bool) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("input %s is not a directory", root)
	}

	var files []string
	if recursive {
		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if isJPEG(path) {
				files = append(files, path)
			}
			return nil
		})
	} else {
		entries, err := os.ReadDir(root)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(root, entry.Name())
			if isJPEG(path) {
				files = append(files, path)
			}
		}
	}
	return files, err
}

func isJPEG(path string) bool {
	lower := strings.ToLower(filepath.Ext(path))
	return lower == ".jpg" || lower == ".jpeg"
}
