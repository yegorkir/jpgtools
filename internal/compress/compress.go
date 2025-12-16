package compress

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/yegorkir/jpgtools/internal/common"
	"github.com/yegorkir/jpgtools/internal/imageutil"
	"github.com/yegorkir/jpgtools/internal/mozjpeg"
)

type options struct {
	Input          string
	Output         string
	Recursive      bool
	Overwrite      bool
	DryRun         bool
	TargetBytes    int64
	InitialQuality int
	MinQuality     int
	QualityStep    int
	Bounds         imageutil.ResizeBounds
}

func Run(args []string) error {
	fs := flag.NewFlagSet("compress", flag.ContinueOnError)
	input := fs.String("input", ".", "Directory with source JPEGs.")
	fs.StringVar(input, "i", ".", "Directory with source JPEGs.")
	output := fs.String("output", "", "Destination directory (default: ./output_YYMMDDhhmm).")
	fs.StringVar(output, "o", "", "Destination directory (default: ./output_YYMMDDhhmm).")
	targetKB := fs.Int("target-kb", 300, "Maximum file size in kilobytes.")
	maxKB := fs.Int("max-kb", 0, "Alias for --target-kb.")
	initialQuality := fs.Int("initial-quality", 85, "Starting mozjpeg quality.")
	minQuality := fs.Int("min-quality", 55, "Minimum mozjpeg quality.")
	qualityStep := fs.Int("quality-step", 5, "Quality decrement between attempts.")
	maxWidth := fs.Int("max-width", 2380, "Maximum width in pixels.")
	maxHeight := fs.Int("max-height", 1600, "Maximum height in pixels.")
	minWidth := fs.Int("min-width", 1290, "Minimum width in pixels.")
	minHeight := fs.Int("min-height", 800, "Minimum height in pixels.")
	recursive := fs.Bool("recursive", false, "Recurse into subdirectories.")
	overwrite := fs.Bool("overwrite", false, "Overwrite files in the output directory.")
	dryRun := fs.Bool("dry-run", false, "Preview work without touching files.")

	if err := fs.Parse(args); err != nil {
		return err
	}

	target := *targetKB
	if *maxKB > 0 {
		target = *maxKB
	}
	if target <= 0 {
		return fmt.Errorf("target kilobytes must be positive")
	}
	if *initialQuality <= 0 || *initialQuality > 100 {
		return fmt.Errorf("initial quality must be between 1 and 100")
	}
	if *minQuality <= 0 || *minQuality > *initialQuality {
		return fmt.Errorf("min quality must be between 1 and initial quality")
	}
	if *qualityStep <= 0 {
		return fmt.Errorf("quality step must be positive")
	}

	bounds := imageutil.ResizeBounds{
		MinWidth:  *minWidth,
		MinHeight: *minHeight,
		MaxWidth:  *maxWidth,
		MaxHeight: *maxHeight,
	}
	if err := bounds.Validate(); err != nil {
		return err
	}

	opt := options{
		Input:          *input,
		Recursive:      *recursive,
		Overwrite:      *overwrite,
		DryRun:         *dryRun,
		TargetBytes:    int64(target) * 1024,
		InitialQuality: *initialQuality,
		MinQuality:     *minQuality,
		QualityStep:    *qualityStep,
		Bounds:         bounds,
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
		fmt.Printf("Using embedded mozjpeg (%s) cached in %s.\n", mozjpeg.Version, filepath.Dir(tc.CJPEG))
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
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	imgInfo, err := imageutil.LoadAndResize(src, opt.Bounds)
	if err != nil {
		return err
	}

	note := imageutil.FormatDimensionNote(imgInfo.Original, imgInfo.Processed, opt.Bounds)

	if opt.DryRun {
		fmt.Printf("[DRY] %s -> %s (%s) target=%dKB quality=%d..%d step=%d\n",
			filepath.Base(src),
			dest,
			note,
			opt.TargetBytes/1024,
			opt.InitialQuality,
			opt.MinQuality,
			opt.QualityStep,
		)
		return nil
	}

	ppmPath, err := imageutil.WritePPM(imgInfo.Image)
	if err != nil {
		return fmt.Errorf("write ppm: %w", err)
	}
	defer os.Remove(ppmPath)

	finalQuality, finalSize, label, err := runQualityLoop(ctx, tc, ppmPath, dest, opt)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] %s -> %s (%s) q=%d size=%.1fKB\n",
		label,
		filepath.Base(src),
		dest,
		note,
		finalQuality,
		float64(finalSize)/1024,
	)
	return nil
}

func runQualityLoop(ctx context.Context, tc *mozjpeg.Toolchain, ppmPath, dest string, opt options) (int, int64, string, error) {
	bestPath := dest + ".best"
	defer os.Remove(bestPath)
	var bestSize int64 = math.MaxInt64
	var bestQuality int

	for quality := opt.InitialQuality; quality >= opt.MinQuality; quality -= opt.QualityStep {
		attempt := fmt.Sprintf("%s.q%d", dest, quality)
		size, err := mozjpeg.EncodePPM(ctx, tc, ppmPath, attempt, mozjpeg.EncodeOptions{Quality: quality})
		if err != nil {
			return 0, 0, "", err
		}
		if size <= opt.TargetBytes {
			os.Remove(dest)
			if err := os.Rename(attempt, dest); err != nil {
				return 0, 0, "", err
			}
			os.Remove(bestPath)
			return quality, size, "OK", nil
		}
		if size < bestSize {
			os.Remove(bestPath)
			if err := os.Rename(attempt, bestPath); err != nil {
				return 0, 0, "", err
			}
			bestSize = size
			bestQuality = quality
		} else {
			os.Remove(attempt)
		}
	}

	if bestSize == math.MaxInt64 {
		return 0, 0, "", fmt.Errorf("failed to encode %s", dest)
	}
	os.Remove(dest)
	if err := os.Rename(bestPath, dest); err != nil {
		return 0, 0, "", err
	}
	return bestQuality, bestSize, "MAXED", nil
}
