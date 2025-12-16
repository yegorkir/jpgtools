package common

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func ResolveOutputDir(requested string) (string, error) {
	if requested != "" {
		return requested, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(cwd, time.Now().UTC().Format("output_0601021504")), nil
}

func EnsureOutputDir(path string, overwrite, dryRun bool) error {
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("output path %s exists and is not a directory", path)
		}
		if !overwrite && !dryRun {
			return fmt.Errorf("output directory %s already exists (use --overwrite)", path)
		}
	} else if os.IsNotExist(err) {
		if dryRun {
			return nil
		}
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	} else {
		return err
	}
	return nil
}
