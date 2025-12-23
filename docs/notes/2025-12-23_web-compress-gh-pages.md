# Web compress app for GitHub Pages

## Summary
- Added a static browser UI for the `compress` workflow so the tool can run locally in the user's browser.
- Implemented JPEG resizing + quality loop with Canvas, mirroring CLI bounds and quality step logic.

## Tradeoffs
- Uses browser JPEG encoder instead of mozjpeg, so output size/quality may differ from CLI.
- Downloads are per-file (no ZIP) to keep the app dependency-free.

## Verification
- Manual: open `docs/index.html`, drop a few JPEGs, confirm size/quality loop and downloads.
