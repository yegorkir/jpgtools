# Embedded mozjpeg toolchain

The archive under `assets/` was produced from `/opt/homebrew/opt/mozjpeg/bin` on macOS arm64.
It contains the unmodified `cjpeg`, `djpeg`, and `jpegtran` binaries provided by the mozjpeg
project (https://github.com/mozilla/mozjpeg). Refer to mozjpeg's upstream license for the
exact terms. The archive is bundled to give the Go CLI a self-contained JPEG encoder pipeline.
