package mozjpeg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type EncodeOptions struct {
	Quality int
}

func EncodePPM(ctx context.Context, tc *Toolchain, ppmPath, destination string, opts EncodeOptions) (int64, error) {
	if tc == nil {
		return 0, fmt.Errorf("toolchain is nil")
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return 0, err
	}

	tmp := destination + ".tmp"
	defer os.Remove(tmp)

	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	in, err := os.Open(ppmPath)
	if err != nil {
		return 0, err
	}
	defer in.Close()

	cmd := exec.CommandContext(
		ctx,
		tc.CJPEG,
		"-quality", fmt.Sprintf("%d", opts.Quality),
		"-optimize",
		"-progressive",
	)
	cmd.Stdin = in
	cmd.Stdout = out
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("cjpeg failed: %w (%s)", err, stderr.String())
	}

	if err := out.Close(); err != nil {
		return 0, err
	}
	if err := os.Rename(tmp, destination); err != nil {
		return 0, err
	}

	info, err := os.Stat(destination)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func CopyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
