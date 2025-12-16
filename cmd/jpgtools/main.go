package main

import (
	"fmt"
	"os"

	"jpgtools/internal/compress"
	"jpgtools/internal/overlay"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "compress":
		err = compress.Run(args)
	case "overlay":
		err = overlay.Run(args)
	case "help", "-h", "--help":
		printUsage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", cmd)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`jpgtools — portable JPEG helper

Usage:
  jpgtools <command> [options]

Commands:
  compress   Recompress JPEGs to hit a target size, mirroring compress_jpgs.py.
  overlay    Apply a semi-transparent black overlay to every JPEG (apply_black_overlay.py).

Run "jpgtools <command> -h" for command-specific options.
`)
}
