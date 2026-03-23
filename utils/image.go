package utils

// ImagePreset defines resize configuration for a named preset.
type ImagePreset struct {
	Width  int    // Width in pixels
	Height int    // Height in pixels (0 for automatic height based on aspect ratio)
	Fit    string // "contain" or "cover"
}
