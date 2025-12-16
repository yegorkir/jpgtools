package imageutil

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"math"
	"os"
)

type ImageInfo struct {
	Image     *image.NRGBA
	Original  [2]int
	Processed [2]int
}

func LoadAndResize(path string, bounds ResizeBounds) (*ImageInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg, err := jpeg.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}

	img := toNRGBA(cfg)
	original := [2]int{img.Bounds().Dx(), img.Bounds().Dy()}
	processed := original

	scale := DetermineScaleFactor(original[0], original[1], bounds)
	if math.Abs(scale-1) > 1e-3 {
		w := max(1, int(math.Round(float64(original[0])*scale)))
		h := max(1, int(math.Round(float64(original[1])*scale)))
		img = resizeBilinear(img, w, h)
		processed = [2]int{w, h}
	}

	return &ImageInfo{
		Image:     img,
		Original:  original,
		Processed: processed,
	}, nil
}

func WritePPM(img *image.NRGBA) (string, error) {
	tmp, err := os.CreateTemp("", "jpgtools-*.ppm")
	if err != nil {
		return "", err
	}

	bw := bufio.NewWriter(tmp)
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	if _, err := fmt.Fprintf(bw, "P6\n%d %d\n255\n", w, h); err != nil {
		tmp.Close()
		return "", err
	}

	row := make([]byte, w*3)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.NRGBAModel.Convert(img.NRGBAAt(x, y)).(color.NRGBA)
			row[x*3+0] = c.R
			row[x*3+1] = c.G
			row[x*3+2] = c.B
		}
		if _, err := bw.Write(row); err != nil {
			tmp.Close()
			return "", err
		}
	}

	if err := bw.Flush(); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	return tmp.Name(), nil
}

func toNRGBA(src image.Image) *image.NRGBA {
	if nrgba, ok := src.(*image.NRGBA); ok && nrgba.Stride == nrgba.Rect.Dx()*4 {
		return nrgba
	}
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}

func resizeBilinear(src *image.NRGBA, width, height int) *image.NRGBA {
	if width == src.Rect.Dx() && height == src.Rect.Dy() {
		return src
	}
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	srcBounds := src.Bounds()
	sW := srcBounds.Dx()
	sH := srcBounds.Dy()

	for y := 0; y < height; y++ {
		fy := (float64(y)+0.5)*float64(sH)/float64(height) - 0.5
		y0 := clampInt(int(math.Floor(fy)), 0, sH-1)
		y1 := clampInt(y0+1, 0, sH-1)
		wy := fy - float64(y0)

		for x := 0; x < width; x++ {
			fx := (float64(x)+0.5)*float64(sW)/float64(width) - 0.5
			x0 := clampInt(int(math.Floor(fx)), 0, sW-1)
			x1 := clampInt(x0+1, 0, sW-1)
			wx := fx - float64(x0)

			c00 := src.NRGBAAt(x0+srcBounds.Min.X, y0+srcBounds.Min.Y)
			c10 := src.NRGBAAt(x1+srcBounds.Min.X, y0+srcBounds.Min.Y)
			c01 := src.NRGBAAt(x0+srcBounds.Min.X, y1+srcBounds.Min.Y)
			c11 := src.NRGBAAt(x1+srcBounds.Min.X, y1+srcBounds.Min.Y)

			dst.SetNRGBA(
				x,
				y,
				blendBilinear(c00, c10, c01, c11, wx, wy),
			)
		}
	}
	return dst
}

func blendBilinear(c00, c10, c01, c11 color.NRGBA, wx, wy float64) color.NRGBA {
	r0 := lerpColor(c00, c10, wx)
	r1 := lerpColor(c01, c11, wx)
	return lerpColor(r0, r1, wy)
}

func lerpColor(a, b color.NRGBA, t float64) color.NRGBA {
	return color.NRGBA{
		R: uint8(lerp(float64(a.R), float64(b.R), t)),
		G: uint8(lerp(float64(a.G), float64(b.G), t)),
		B: uint8(lerp(float64(a.B), float64(b.B), t)),
		A: uint8(lerp(float64(a.A), float64(b.A), t)),
	}
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func clampInt(v, minV, maxV int) int {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
