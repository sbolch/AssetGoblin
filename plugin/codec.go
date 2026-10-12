package plugin

import (
	"assetgoblin/image"
	"context"
	"fmt"
	"mime"
	"os"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

type imageCodec struct {
	plugin         *script
	encode, decode *lua.LFunction
	mime           string
	extension      string
}

// ImageCodecs returns an independent codec registry for the core image service.
func (m *Manager) ImageCodecs() map[string]image.Codec {
	result := map[string]image.Codec{}
	for name, codec := range m.codecs {
		result[name] = codec
	}
	return result
}
func (c *imageCodec) Revision() string { return c.plugin.revision }
func (c *imageCodec) MIMEType() string { return c.mime }
func (c *imageCodec) Decode(ctx context.Context, file *os.File, identity string) (string, error) {
	return c.run(ctx, c.decode, file, identity, "png")
}
func (c *imageCodec) Encode(ctx context.Context, file *os.File, identity string) (string, error) {
	return c.run(ctx, c.encode, file, identity, c.extension)
}
func (c *imageCodec) run(ctx context.Context, fn *lua.LFunction, file *os.File, identity, requiredExtension string) (string, error) {
	plugin := c.plugin
	scriptCtx, cancel := context.WithTimeout(ctx, scriptTimeout)
	defer cancel()
	select {
	case plugin.gate <- struct{}{}:
	case <-scriptCtx.Done():
		return "", scriptCtx.Err()
	}
	var plan *processPlan
	err := func() error {
		defer func() { <-plugin.gate }()
		if plugin.closed {
			return fmt.Errorf("plugin closed")
		}
		state := plugin.state
		state.SetContext(scriptCtx)
		defer state.RemoveContext()
		top := state.GetTop()
		defer state.SetTop(top)
		request := state.NewTable()
		request.RawSetString("path", lua.LString(identity))
		if err := state.CallByParam(lua.P{Fn: fn, NRet: 1, Protect: true}, request); err != nil {
			return err
		}
		var err error
		plan, err = parseProcess(state.Get(-1))
		return err
	}()
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
func (s *script) loadCodecs(value lua.LValue) error {
	if value == lua.LNil {
		return nil
	}
	table, ok := value.(*lua.LTable)
	if !ok {
		return fmt.Errorf("image_codecs must be a table")
	}
	var loadErr error
	table.ForEach(func(key, value lua.LValue) {
		name, ok := key.(lua.LString)
		if !ok || len(name) == 0 || len(name) > 16 || strings.Trim(string(name), "abcdefghijklmnopqrstuvwxyz0123456789") != "" {
			loadErr = fmt.Errorf("invalid image codec name")
			return
		}
		spec, ok := value.(*lua.LTable)
		if !ok {
			loadErr = fmt.Errorf("image codec must be a table")
			return
		}
		encode, eOK := spec.RawGetString("encode").(*lua.LFunction)
		decode, dOK := spec.RawGetString("decode").(*lua.LFunction)
		mimeValue, mOK := spec.RawGetString("mime_type").(lua.LString)
		mediaType, _, mimeErr := mime.ParseMediaType(string(mimeValue))
		if !eOK || !dOK || !mOK || mimeErr != nil || strings.ContainsAny(string(mimeValue), "\r\n\x00") || !strings.HasPrefix(mediaType, "image/") {
			loadErr = fmt.Errorf("image codec needs encode/decode functions and image MIME type")
			return
		}
		s.codecs[string(name)] = &imageCodec{plugin: s, encode: encode, decode: decode, mime: string(mimeValue), extension: string(name)}
	})
	return loadErr
}
