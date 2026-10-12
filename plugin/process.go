package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const maxInputSize = 128 << 20
const maxOutputSize = 32 << 20

var worker = make(chan struct{}, 1)

// processPlan is generic: scripts define executable, arguments and output extension.
type processPlan struct {
	Tool      string
	Args      []string
	Extension string
}

func (p processPlan) validate() error {
	if !validToolName(p.Tool) {
		return fmt.Errorf("invalid process tool")
	}
	if len(p.Extension) == 0 || len(p.Extension) > 16 || strings.Trim(p.Extension, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789") != "" {
		return fmt.Errorf("invalid output extension")
	}
	if len(p.Args) == 0 || len(p.Args) > 256 {
		return fmt.Errorf("process requires 1..256 arguments")
	}
	input, output := false, false
	total := 0
	for _, arg := range p.Args {
		if strings.ContainsRune(arg, 0) {
			return fmt.Errorf("process argument contains NUL")
		}
		total += len(arg)
		input = input || arg == "{input}"
		output = output || arg == "{output}"
	}
	if !input || !output || total > 16384 {
		return fmt.Errorf("process needs {input}/{output} arguments and at most 16 KiB of arguments")
	}
	return nil
}

func validToolName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, "/\\:\x00") && !strings.ContainsAny(name, " \t\r\n")
}

// runProcess stages root-scoped input and runs a plugin-defined tool without a shell.
// The host controls cancellation, size limits, cache publication and cleanup.
func runProcess(parent context.Context, source *os.File, identity, revision, cacheDir string, plan processPlan) (string, error) {
	if err := plan.validate(); err != nil {
		return "", err
	}
	info, err := source.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() > maxInputSize {
		return "", fmt.Errorf("source must be a regular file no larger than 128 MiB")
	}
	if cacheDir == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			return "", err
		}
		cacheDir = filepath.Join(base, "assetgoblin", "plugins")
	}
	encoded, _ := json.Marshal(struct {
		Version            int
		Identity, Revision string
		Size, Modified     int64
		Plan               processPlan
	}{1, identity, revision, info.Size(), info.ModTime().UnixNano(), plan})
	key := fmt.Sprintf("%x", sha256.Sum256(encoded))
	output := filepath.Join(cacheDir, key+"."+plan.Extension)
	if stat, err := os.Stat(output); err == nil && stat.Mode().IsRegular() {
		return output, nil
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	select {
	case worker <- struct{}{}:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	defer func() { <-worker }()
	// Recheck after acquiring the worker: identical simultaneous requests reuse output.
	if stat, err := os.Stat(output); err == nil && stat.Mode().IsRegular() {
		return output, nil
	}
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return "", err
	}
	input, err := os.CreateTemp(cacheDir, "input-*"+filepath.Ext(identity))
	if err != nil {
		return "", err
	}
	inputPath := input.Name()
	defer os.Remove(inputPath)
	_, copyErr := io.Copy(input, &contextReader{ctx: ctx, reader: io.LimitReader(source, maxInputSize+1)})
	closeErr := input.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	stat, err := os.Stat(inputPath)
	if err != nil || stat.Size() > maxInputSize {
		return "", fmt.Errorf("source exceeded input limit")
	}
	temp, err := os.CreateTemp(cacheDir, "output-*."+plan.Extension)
	if err != nil {
		return "", err
	}
	tempPath := temp.Name()
	temp.Close()
	defer os.Remove(tempPath)
	args := append([]string(nil), plan.Args...)
	for i, arg := range args {
		if arg == "{input}" {
			args[i] = inputPath
		}
		if arg == "{output}" {
			args[i] = tempPath
		}
	}
	cmd := exec.CommandContext(ctx, plan.Tool, args...)
	cmd.WaitDelay = time.Second
	// Avoid buffering tool logs, whose size can depend on malformed input.
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := runBounded(ctx, cancel, cmd, tempPath); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", fmt.Errorf("plugin tool %s unavailable: %w", plan.Tool, err)
		}
		return "", fmt.Errorf("plugin tool %s failed: %w", plan.Tool, err)
	}
	stat, err = os.Stat(tempPath)
	if err != nil {
		return "", err
	}
	if stat.Size() == 0 || stat.Size() >= maxOutputSize {
		return "", fmt.Errorf("output is empty or exceeds 32 MiB")
	}
	if err := os.Rename(tempPath, output); err != nil {
		return "", err
	}
	return output, nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

// Generic file size supervision also applies to tools without their own output cap.
func runBounded(ctx context.Context, cancel context.CancelFunc, cmd *exec.Cmd, output string) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		case <-ticker.C:
			if info, err := os.Stat(output); err == nil && info.Size() >= maxOutputSize {
				cancel()
				<-done
				return fmt.Errorf("output exceeded 32 MiB")
			}
		case <-ctx.Done():
			<-done
			return ctx.Err()
		}
	}
}

// A pipeline shares one deadline, including worker queue time, across all stages.
func runPipeline(ctx context.Context, source *os.File, identity, revision string, plans []processPlan, tools map[string]bool) (string, error) {
	for _, plan := range plans {
		if !tools[plan.Tool] {
			return "", fmt.Errorf("undeclared tool %s", plan.Tool)
		}
		if err := plan.validate(); err != nil {
			return "", err
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var output string
	for i, plan := range plans {
		input := source
		if i > 0 {
			var err error
			input, err = os.Open(output)
			if err != nil {
				return "", err
			}
			identity = output
		}
		next, err := runProcess(ctx, input, identity, revision, "", plan)
		if i > 0 {
			input.Close()
		}
		if err != nil {
			return "", err
		}
		output = next
	}
	return output, nil
}
