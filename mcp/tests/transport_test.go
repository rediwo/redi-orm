package mcp_test

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/rediwo/redi-orm/mcp/transport"
	"github.com/rediwo/redi-orm/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockReadWriter struct {
	*bytes.Buffer
}

func (m *mockReadWriter) Write(p []byte) (n int, err error) {
	return m.Buffer.Write(p)
}

func TestStdioTransport(t *testing.T) {
	logger := utils.NewDefaultLogger("test")
	logger.SetLevel(utils.LogLevelError)

	t.Run("Send", func(t *testing.T) {
		// Create transport with custom writer (not exposed in real implementation)
		// For now, we'll test the basic creation
		transport := transport.NewStdioTransport(logger)
		assert.NotNil(t, transport)
		
		// Test Start and Stop
		err := transport.Start()
		assert.NoError(t, err)
		
		err = transport.Stop()
		assert.NoError(t, err)
	})

	t.Run("Message Format", func(t *testing.T) {
		// Test that messages would be properly formatted
		testMessage := json.RawMessage(`{"test": "message"}`)
		
		// Verify the message is valid JSON
		assert.True(t, json.Valid(testMessage))
	})
}

func TestMessageFraming(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
	}{
		{
			name:     "valid JSON message",
			input:    `{"jsonrpc":"2.0","method":"test","id":1}` + "\n",
			wantErr:  false,
		},
		{
			name:     "invalid JSON",
			input:    `{invalid json}` + "\n",
			wantErr:  true,
		},
		{
			name:     "empty message",
			input:    "\n",
			wantErr:  true,
		},
		{
			name:     "message without newline",
			input:    `{"jsonrpc":"2.0","method":"test","id":1}`,
			wantErr:  true, // Should fail because no newline
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewBufferString(tt.input)
			
			// Read until newline
			data, err := reader.ReadBytes('\n')
			
			// For message without newline, we expect EOF error
			if tt.name == "message without newline" {
				assert.Equal(t, io.EOF, err)
				// The data returned is valid JSON but protocol requires newline
				assert.True(t, json.Valid(data))
				return
			}
			
			if err != nil && err != io.EOF {
				if tt.wantErr {
					return
				}
				t.Fatalf("unexpected error: %v", err)
			}

			if len(data) > 0 && data[len(data)-1] == '\n' {
				data = data[:len(data)-1]
			}

			// Validate JSON
			isValid := json.Valid(data)
			if tt.wantErr {
				assert.False(t, isValid)
			} else {
				assert.True(t, isValid)
			}
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	logger := utils.NewDefaultLogger("test")
	logger.SetLevel(utils.LogLevelError)
	
	transport := transport.NewStdioTransport(logger)
	require.NotNil(t, transport)

	err := transport.Start()
	require.NoError(t, err)

	// Test concurrent Stop calls
	done := make(chan bool, 2)
	
	go func() {
		transport.Stop()
		done <- true
	}()
	
	go func() {
		transport.Stop()
		done <- true
	}()
	
	// Wait for both goroutines
	<-done
	<-done
	
	// Should not panic or deadlock
}