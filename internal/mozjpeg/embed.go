package mozjpeg

import (
	_ "embed"
	"fmt"
	"runtime"
)

const Version = "mozjpeg-4.1.4-embedded"

//go:embed assets/darwin-arm64/mozjpeg.tar.gz
var darwinArm64Archive []byte

func assetForCurrentPlatform() ([]byte, string, error) {
	key := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	switch key {
	case "darwin-arm64":
		return darwinArm64Archive, key, nil
	default:
		return nil, "", fmt.Errorf("no embedded mozjpeg toolchain for %s", key)
	}
}
