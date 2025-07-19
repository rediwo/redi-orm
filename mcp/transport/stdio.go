package transport

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/rediwo/redi-orm/utils"
)

// StdioTransport implements Transport interface for standard I/O communication
type StdioTransport struct {
	reader  *bufio.Reader
	writer  io.Writer
	logger  utils.Logger
	mu      sync.Mutex
	closed  bool
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport(logger utils.Logger) *StdioTransport {
	return &StdioTransport{
		reader: bufio.NewReader(os.Stdin),
		writer: os.Stdout,
		logger: logger,
	}
}

// Start initializes the stdio transport
func (t *StdioTransport) Start() error {
	t.logger.Debug("Starting stdio transport")
	return nil
}

// Stop closes the stdio transport
func (t *StdioTransport) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.closed = true
	t.logger.Debug("Stopped stdio transport")
	return nil
}

// Send writes a message to stdout
func (t *StdioTransport) Send(message json.RawMessage) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if t.closed {
		return fmt.Errorf("transport is closed")
	}

	// Write message with newline delimiter
	if _, err := t.writer.Write(message); err != nil {
		return err
	}
	if _, err := t.writer.Write([]byte("\n")); err != nil {
		return err
	}

	// Flush if writer supports it
	if flusher, ok := t.writer.(interface{ Flush() error }); ok {
		if err := flusher.Flush(); err != nil {
			return err
		}
	}

	t.logger.Debug("Sent message with size: %d", len(message))
	return nil
}

// Receive reads a message from stdin
func (t *StdioTransport) Receive() (json.RawMessage, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, fmt.Errorf("transport is closed")
	}
	t.mu.Unlock()

	// Read line from stdin
	line, err := t.reader.ReadBytes('\n')
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("EOF")
		}
		return nil, err
	}

	// Trim newline
	line = line[:len(line)-1]

	// Validate JSON
	var msg json.RawMessage = line
	if !json.Valid(msg) {
		return nil, fmt.Errorf("invalid JSON message")
	}

	t.logger.Debug("Received message with size: %d", len(msg))
	return msg, nil
}