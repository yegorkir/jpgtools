package imageutil

import "image"

const defaultOverlayAlpha = 0.2

func ApplyBlackOverlay(img *image.NRGBA, alpha float64) {
	if alpha <= 0 {
		return
	}
	if alpha > 1 {
		alpha = 1
	}
	scale := 1 - alpha
	pix := img.Pix
	for i := 0; i < len(pix); i += 4 {
		pix[i+0] = uint8(float64(pix[i+0]) * scale)
		pix[i+1] = uint8(float64(pix[i+1]) * scale)
		pix[i+2] = uint8(float64(pix[i+2]) * scale)
	}
}
