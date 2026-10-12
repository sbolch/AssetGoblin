package image

import (
	"context"
	"os"
)

// Codec bridges an extension's format to/from PNG. All encoding logic belongs
// to the extension; the core keeps its existing presets and transformations.
type Codec interface {
	Decode(context.Context, *os.File, string) (string, error)
	Encode(context.Context, *os.File, string) (string, error)
	Revision() string
	MIMEType() string
}
