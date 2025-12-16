package mozjpeg

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Toolchain struct {
	CJPEG    string
	DJPEG    string
	JPEGTran string
}

func Ensure(ctx context.Context) (*Toolchain, error) {
	bytes, platformKey, err := assetForCurrentPlatform()
	if err != nil {
		return nil, err
	}

	cacheRoot, err := cacheDir()
	if err != nil {
		return nil, err
	}

	targetDir := filepath.Join(cacheRoot, Version, platformKey)
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := ensureExtracted(targetDir, bytes); err != nil {
		return nil, err
	}

	return &Toolchain{
		CJPEG:    filepath.Join(targetDir, "cjpeg"),
		DJPEG:    filepath.Join(targetDir, "djpeg"),
		JPEGTran: filepath.Join(targetDir, "jpegtran"),
	}, nil
}

func cacheDir() (string, error) {
	if dir := os.Getenv("JPGTOOLS_CACHE_DIR"); dir != "" {
		return dir, nil
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "jpgtools"), nil
}

func ensureExtracted(dest string, archive []byte) error {
	if stat, err := os.Stat(filepath.Join(dest, ".ready")); err == nil && !stat.IsDir() {
		return nil
	}

	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}

	if err := untar(bytes.NewReader(archive), dest); err != nil {
		return err
	}

	sentinel := filepath.Join(dest, ".ready")
	return os.WriteFile(sentinel, []byte("ok"), 0o644)
}

func untar(r io.Reader, dest string) error {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("init gzip: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}

		name := filepath.Clean(hdr.Name)
		if strings.Contains(name, "..") {
			return fmt.Errorf("unsafe path in archive: %q", hdr.Name)
		}
		target := filepath.Join(dest, name)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		default:
			continue
		}
	}
	return nil
}
