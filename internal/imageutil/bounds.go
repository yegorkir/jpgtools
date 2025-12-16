package imageutil

import (
	"fmt"
	"math"
	"strings"
)

type ResizeBounds struct {
	MinWidth  int
	MinHeight int
	MaxWidth  int
	MaxHeight int
}

func (b ResizeBounds) Validate() error {
	if b.MinWidth < 0 || b.MinHeight < 0 || b.MaxWidth < 0 || b.MaxHeight < 0 {
		return fmt.Errorf("bounds must be non-negative")
	}
	return nil
}

func DetermineScaleFactor(width, height int, bounds ResizeBounds) float64 {
	if width <= 0 || height <= 0 {
		return 1
	}

	lower := 0.0
	upper := math.Inf(1)

	if bounds.MinWidth > 0 {
		lower = math.Max(lower, float64(bounds.MinWidth)/float64(width))
	}
	if bounds.MinHeight > 0 {
		lower = math.Max(lower, float64(bounds.MinHeight)/float64(height))
	}
	if bounds.MaxWidth > 0 {
		upper = math.Min(upper, float64(bounds.MaxWidth)/float64(width))
	}
	if bounds.MaxHeight > 0 {
		upper = math.Min(upper, float64(bounds.MaxHeight)/float64(height))
	}

	if lower <= 1 && 1 <= upper {
		return 1
	}
	if lower > upper {
		if upper < 1 {
			return upper
		}
		return lower
	}
	if lower > 1 {
		return lower
	}
	if upper < 1 {
		return upper
	}
	return 1
}

func FormatDimensionNote(original, processed [2]int, bounds ResizeBounds) string {
	note := fmt.Sprintf("%dx%d", original[0], original[1])
	if original == processed {
		return note
	}

	note = fmt.Sprintf("%s->%dx%d", note, processed[0], processed[1])
	warnings := make([]string, 0, 2)
	if bounds.MinWidth > 0 && processed[0] < bounds.MinWidth {
		warnings = append(warnings, "below min width")
	}
	if bounds.MinHeight > 0 && processed[1] < bounds.MinHeight {
		warnings = append(warnings, "below min height")
	}
	if bounds.MaxWidth > 0 && processed[0] > bounds.MaxWidth {
		warnings = append(warnings, "above max width")
	}
	if bounds.MaxHeight > 0 && processed[1] > bounds.MaxHeight {
		warnings = append(warnings, "above max height")
	}
	if len(warnings) > 0 {
		note = fmt.Sprintf("%s (%s)", note, strings.Join(warnings, ", "))
	}
	return note
}
