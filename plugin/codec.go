package plugin

import (
	"context"
	"fmt"
	"maps"
	"mime"
	"os"
	"strings"

	"assetgoblin/image"
)

type imageCodec struct {
	plugin         *script
	encode, decode Callback
	mime           string
	extension      string
}

// ImageCodecs returns an independent codec registry for the core image service.
func (m *Manager) ImageCodecs() map[string]image.Codec {
	return maps.Clone(m.codecs)
}
func (c *imageCodec) Revision() string { return c.plugin.revision }
func (c *imageCodec) MIMEType() string { return c.mime }
func (c *imageCodec) Decode(ctx context.Context, file *os.File, identity string) (string, error) {
	return c.run(ctx, c.decode, file, identity, "png")
}
func (c *imageCodec) Encode(ctx context.Context, file *os.File, identity string) (string, error) {
	return c.run(ctx, c.encode, file, identity, c.extension)
}
func (c *imageCodec) run(ctx context.Context, fn Callback, file *os.File, identity, requiredExtension string) (string, error) {
	plugin := c.plugin
	value, err := plugin.invoke(ctx, fn, map[string]any{"path": identity})
	if err != nil {
		return "", err
	}
	plan, err := parseProcess(value)
	if err != nil {
		return "", err
	}

	if !plugin.tools[plan.Tool] {
		return "", fmt.Errorf("codec requested undeclared tool")
	}
	if requiredExtension != "" && plan.Extension != requiredExtension {
		return "", fmt.Errorf("image codec returned unexpected output extension")
	}
	return runProcess(ctx, file, identity, plugin.revision, "", *plan)
}
func (s *script) loadCodecs(value any) error {
	if value == nil {
		return nil
	}
	table, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("image_codecs must be an object")
	}
	for name, value := range table {
		if len(name) == 0 || len(name) > 16 || strings.Trim(name, "abcdefghijklmnopqrstuvwxyz0123456789") != "" {
			return fmt.Errorf("invalid image codec name")
		}
		spec, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("image codec must be an object")
		}
		encode, eOK := spec["encode"].(Callback)
		decode, dOK := spec["decode"].(Callback)
		mimeValue, mOK := spec["mime_type"].(string)
		mediaType, _, mimeErr := mime.ParseMediaType(mimeValue)
		if !eOK || !dOK || !mOK || mimeErr != nil || strings.ContainsAny(mimeValue, "\r\n\x00") || !strings.HasPrefix(mediaType, "image/") {
			return fmt.Errorf("image codec needs encode/decode functions and image MIME type")
		}
		s.codecs[name] = &imageCodec{plugin: s, encode: encode, decode: decode, mime: mimeValue, extension: name}
	}
	return nil
}
