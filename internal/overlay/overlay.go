package overlay

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"jpgtools/internal/common"
	"jpgtools/internal/imageutil"
	"jpgtools/internal/mozjpeg"
)

type options struct {
	Input     string
	Output    string
	Recursive bool
	Overwrite bool
	DryRun    bool
	Quality   int
	Alpha     float64
}

func Run(args []string) error {
	fs := flag.NewFlagSet("overlay", flag.ContinueOnError)
	input := fs.String("input", ".", "Directory with source JPEGs.")
	fs.StringVar(input, "i", ".", "Directory with source JPEGs.")
	output := fs.String("output", "", "Destination directory (default: ./output_YYMMDDhhmm).")
	fs.StringVar(output, "o", "", "Destination directory (default: ./output_YYMMDDhhmm).")
	recursive := fs.Bool("recursive", false, "Recurse into subdirectories.")
	overwrite := fs.Bool("overwrite", false, "Overwrite files in the output directory.")
	dryRun := fs.Bool("dry-run", false, "Preview work without touching files.")
	quality := fs.Int("quality", 95, "mozjpeg quality for the re-encoded image.")
	alpha := fs.Float64("alpha", 0.2, "Overlay opacity (0..1).")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *quality <= 0 || *quality > 100 {
		return fmt.Errorf("quality must be between 1 and 100")
	}
	if *alpha < 0 || *alpha > 1 {
		return fmt.Errorf("alpha must be between 0 and 1")
	}

	opt := options{
		Input:     *input,
		Recursive: *recursive,
		Overwrite: *overwrite,
		DryRun:    *dryRun,
		Quality:   *quality,
		Alpha:     *alpha,
	}

	out, err := common.ResolveOutputDir(*output)
	if err != nil {
		return err
	}
	opt.Output = out

	if err := common.EnsureOutputDir(out, opt.Overwrite, opt.DryRun); err != nil {
		return err
	}

	files, err := common.CollectJPEGs(opt.Input, opt.Recursive)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		fmt.Printf("No JPEG files found in %s.\n", opt.Input)
		return nil
	}

	ctx := context.Background()
	var tc *mozjpeg.Toolchain
	if !opt.DryRun {
		tc, err = mozjpeg.Ensure(ctx)
		if err != nil {
			return fmt.Errorf("prepare mozjpeg: %w", err)
		}
		fmt.Printf("Using embedded mozjpeg (%s).\n", mozjpeg.Version)
	} else {
		fmt.Println("Running in dry-run mode. No files will be written.")
	}

	start := time.Now()
	for _, src := range files {
		rel, err := filepath.Rel(opt.Input, src)
		if err != nil {
			rel = filepath.Base(src)
		}
		dest := filepath.Join(opt.Output, rel)
		if err := processFile(ctx, tc, src, dest, opt); err != nil {
			fmt.Printf("[ERROR] %s: %v\n", src, err)
		}
	}

	fmt.Printf("Done in %s.\n", time.Since(start).Truncate(time.Millisecond))
	return nil
}

func processFile(ctx context.Context, tc *mozjpeg.Toolchain, src, dest string, opt options) error {
	if _, err := os.Stat(dest); err == nil && !opt.Overwrite && !opt.DryRun {
		fmt.Printf("[SKIP] %s exists (use --overwrite).\n", dest)
		return nil
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	imgInfo, err := imageutil.LoadAndResize(src, imageutil.ResizeBounds{})
	if err != nil {
		return err
	}

	if opt.DryRun {
		fmt.Printf("[DRY] overlay %s -> %s (%dx%d) alpha=%.2f quality=%d\n",
			filepath.Base(src),
			dest,
			imgInfo.Original[0],
			imgInfo.Original[1],
			opt.Alpha,
			opt.Quality,
		)
		return nil
	}

	imageutil.ApplyBlackOverlay(imgInfo.Image, opt.Alpha)

	ppmPath, err := imageutil.WritePPM(imgInfo.Image)
	if err != nil {
		return fmt.Errorf("write ppm: %w", err)
	}
	defer os.Remove(ppmPath)

	size, err := mozjpeg.EncodePPM(ctx, tc, ppmPath, dest, mozjpeg.EncodeOptions{Quality: opt.Quality})
	if err != nil {
		return err
	}

	fmt.Printf("[OK] %s -> %s (%dx%d) size=%.1fKB\n",
		filepath.Base(src),
		dest,
		imgInfo.Original[0],
		imgInfo.Original[1],
		float64(size)/1024,
	)
	return nil
}
