package utils

// ImagePreset defines resize configuration for a named preset.
type ImagePreset struct {
	Width      int      // Width in pixels
	Height     int      // Height in pixels (0 for automatic height based on aspect ratio)
	Fit        string   // "contain" or "cover"
	Rotate     int      // Rotation in degrees (0, 90, 180, 270)
	Flip       string   // "horizontal", "vertical", or "both"
	Crop       string   // Crop region: "top-left", "top", "top-right", "left", "center", "right", "bottom-left", "bottom", "bottom-right"
	Brightness float64  // Brightness adjustment (-100 to 100)
	Contrast   float64  // Contrast adjustment (-100 to 100)
	Gamma      float64  // Gamma adjustment (0.1 to 10.0)
	Filters    []string // Image filters to apply in order
}
