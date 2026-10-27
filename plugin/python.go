package plugin

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

//go:embed python_runner.py
var pythonRunner string

const maxPythonMessage = 4 << 20

type pythonWorker struct {
	cmd    *exec.Cmd
	input  io.WriteCloser
	output io.ReadCloser
	reader *bufio.Scanner
	done   chan error
	once   sync.Once
}

type pythonReply struct {
	Value    any      `json:"value"`
	Error    string   `json:"error"`
	Parents  []string `json:"parents"`
	Revision string   `json:"revision"`
}

func (w *pythonWorker) close() {
	w.once.Do(func() {
		// Closing pipes unblocks pending protocol writes/reads as well as killing code.
		w.input.Close()
		w.output.Close()
		w.cmd.Process.Kill()
		<-w.done
	})
}

// exchange is called only while holding the plugin gate (or during startup).
func (w *pythonWorker) exchange(ctx context.Context, message any) (pythonReply, error) {
	encoded, err := json.Marshal(message)
	if err != nil {
		return pythonReply{}, err
	}
	if len(encoded) > maxPythonMessage {
		return pythonReply{}, fmt.Errorf("python request exceeds 4 MiB")
	}
	type result struct {
		reply pythonReply
		err   error
	}
	finished := make(chan result, 1)
	go func() {
		if _, err := w.input.Write(append(encoded, '\n')); err != nil {
			finished <- result{err: err}
			return
		}
		if !w.reader.Scan() {
			err := w.reader.Err()
			if err == nil {
				err = io.EOF
			}
			finished <- result{err: fmt.Errorf("python worker protocol: %w", err)}
			return
		}
		var reply pythonReply
		err := json.Unmarshal(w.reader.Bytes(), &reply)
		if err == nil && reply.Error != "" {
			err = fmt.Errorf("python plugin: %s", reply.Error)
		}
		finished <- result{reply, err}
	}()
	select {
	case result := <-finished:
		if result.err != nil && result.reply.Error == "" {
			w.close()
		}
		return result.reply, result.err
	case <-ctx.Done():
		w.close()
		<-finished
		return pythonReply{}, ctx.Err()
	}
}

func loadPython(filename string, catalog map[string]string, executable string) (_ *script, _ []RouteHandler, err error) {
	if executable == "" {
		for _, name := range []string{"python3", "python"} {
			if path, e := exec.LookPath(name); e == nil {
				executable = path
				break
			}
		}
		if executable == "" {
			return nil, nil, fmt.Errorf("python 3.10+ is required for .py plugins; install Python or configure python_executable")
		}
	}
	cmd := exec.Command(executable, "-I", "-u", "-X", "utf8", "-c", pythonRunner)
	cmd.WaitDelay = time.Second
	input, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	output, outputWriter, err := os.Pipe()
	if err != nil {
		input.Close()
		return nil, nil, err
	}
	cmd.Stdout = outputWriter
	cmd.Stderr = io.Discard
	if err = cmd.Start(); err != nil {
		input.Close()
		output.Close()
		outputWriter.Close()
		return nil, nil, fmt.Errorf("start Python: %w", err)
	}
	// Own the read pipe: Cmd.Wait must not close it before a final error is read.
	outputWriter.Close()
	worker := &pythonWorker{cmd: cmd, input: input, output: output, reader: bufio.NewScanner(output), done: make(chan error, 1)}
	worker.reader.Buffer(make([]byte, 4096), maxPythonMessage+1)
	go func() { worker.done <- cmd.Wait() }()
	defer func() {
		if err != nil {
			worker.close()
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), scriptTimeout)
	defer cancel()
	reply, err := worker.exchange(ctx, map[string]any{"filename": filename, "catalog": catalog})
	if err != nil {
		return nil, nil, err
	}
	plugin := newScript(filename)
	plugin.close = worker.close
	plugin.parents, plugin.revision = reply.Parents, reply.Revision
	manifest := worker.bind(reply.Value)
	routes, err := plugin.loadManifest(manifest)
	return plugin, routes, err
}

func (w *pythonWorker) bind(value any) any {
	switch value := value.(type) {
	case map[string]any:
		if key, ok := value["$callback"].(string); ok && len(value) == 1 {
			return Callback(func(ctx context.Context, request map[string]any) (any, error) {
				reply, err := w.exchange(ctx, map[string]any{"callback": key, "request": request})
				return reply.Value, err
			})
		}
		for key, item := range value {
			value[key] = w.bind(item)
		}
	case []any:
		for i, item := range value {
			value[i] = w.bind(item)
		}
	}
	return value
}

// LoadWithPython selects an interpreter, including a virtual environment's Python.
// Directories without Python plugins do not need Python installed.
func LoadWithPython(directory, executable string) (*Manager, error) {
	return load(directory, executable)
}
