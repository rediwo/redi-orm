package registry

import (
	"fmt"
	"sync"
	"testing"

	"github.com/rediwo/redi-orm/types"
)

// Clear registries for testing
func clearRegistries() {
	mu.Lock()
	defer mu.Unlock()
	drivers = make(map[string]DriverFactory)
	uriParsers = make(map[string]types.URIParser)
}

// Mock URIParser implementation
type mockURIParser struct {
	supportedSchemes []string
	driverType      string
	parseFunc       func(uri string) (types.Config, error)
}

func (m *mockURIParser) ParseURI(uri string) (types.Config, error) {
	if m.parseFunc != nil {
		return m.parseFunc(uri)
	}
	return types.Config{}, fmt.Errorf("not supported")
}

func (m *mockURIParser) GetSupportedSchemes() []string {
	return m.supportedSchemes
}

func (m *mockURIParser) GetDriverType() string {
	return m.driverType
}

func TestRegister(t *testing.T) {
	clearRegistries()
	
	tests := []struct {
		name        string
		driverType  string
		factory     DriverFactory
		shouldPanic bool
	}{
		{
			name:       "register new driver",
			driverType: "testdb",
			factory: func(config types.Config) (types.Database, error) {
				return nil, nil
			},
			shouldPanic: false,
		},
		{
			name:       "register duplicate driver",
			driverType: "duplicate",
			factory: func(config types.Config) (types.Database, error) {
				return nil, nil
			},
			shouldPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				// First register the driver
				Register(tt.driverType, tt.factory)
				
				// Then test panic on duplicate
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Register() should panic for duplicate driver")
					}
				}()
			}
			
			Register(tt.driverType, tt.factory)
			
			if !tt.shouldPanic {
				// Verify registration
				factory, err := Get(tt.driverType)
				if err != nil {
					t.Errorf("Get() error = %v", err)
				}
				if factory == nil {
					t.Error("Get() returned nil factory")
				}
			}
		})
	}
}

func TestGet(t *testing.T) {
	clearRegistries()
	
	// Register a test driver
	testFactory := func(config types.Config) (types.Database, error) {
		return nil, nil
	}
	Register("gettest", testFactory)

	tests := []struct {
		name       string
		driverType string
		wantErr    bool
	}{
		{
			name:       "get existing driver",
			driverType: "gettest",
			wantErr:    false,
		},
		{
			name:       "get non-existing driver",
			driverType: "nonexistent",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory, err := Get(tt.driverType)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && factory == nil {
				t.Error("Get() returned nil factory for existing driver")
			}
		})
	}
}

func TestRegisterURIParser(t *testing.T) {
	clearRegistries()
	
	tests := []struct {
		name        string
		driverType  string
		parser      types.URIParser
		shouldPanic bool
	}{
		{
			name:       "register new parser",
			driverType: "testparser",
			parser: &mockURIParser{
				driverType: "testparser",
				supportedSchemes: []string{"test"},
			},
			shouldPanic: false,
		},
		{
			name:       "register duplicate parser",
			driverType: "dupparser",
			parser: &mockURIParser{
				driverType: "dupparser",
				supportedSchemes: []string{"dup"},
			},
			shouldPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				// First register the parser
				RegisterURIParser(tt.driverType, tt.parser)
				
				// Then test panic on duplicate
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("RegisterURIParser() should panic for duplicate parser")
					}
				}()
			}
			
			RegisterURIParser(tt.driverType, tt.parser)
			
			if !tt.shouldPanic {
				// Verify registration
				parser, err := GetURIParser(tt.driverType)
				if err != nil {
					t.Errorf("GetURIParser() error = %v", err)
				}
				if parser == nil {
					t.Error("GetURIParser() returned nil")
				}
			}
		})
	}
}

func TestGetURIParser(t *testing.T) {
	clearRegistries()
	
	// Register a test parser
	testParser := &mockURIParser{
		driverType: "parsertest",
		supportedSchemes: []string{"ptest"},
	}
	RegisterURIParser("parsertest", testParser)

	tests := []struct {
		name       string
		driverType string
		wantErr    bool
	}{
		{
			name:       "get existing parser",
			driverType: "parsertest",
			wantErr:    false,
		},
		{
			name:       "get non-existing parser",
			driverType: "nonexistent",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := GetURIParser(tt.driverType)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetURIParser() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && parser == nil {
				t.Error("GetURIParser() returned nil for existing parser")
			}
		})
	}
}

func TestGetAllURIParsers(t *testing.T) {
	clearRegistries()
	
	// Register some parsers for testing
	parser1 := &mockURIParser{
		driverType: "type1",
		supportedSchemes: []string{"type1"},
	}
	parser2 := &mockURIParser{
		driverType: "type2", 
		supportedSchemes: []string{"type2"},
	}
	
	RegisterURIParser("type1", parser1)
	RegisterURIParser("type2", parser2)
	
	parsers := GetAllURIParsers()
	
	if len(parsers) != 2 {
		t.Errorf("GetAllURIParsers() returned %d parsers, want 2", len(parsers))
	}
	
	// Verify both parsers are present
	if _, ok := parsers["type1"]; !ok {
		t.Error("GetAllURIParsers() missing type1 parser")
	}
	if _, ok := parsers["type2"]; !ok {
		t.Error("GetAllURIParsers() missing type2 parser")
	}
}

func TestParseURI(t *testing.T) {
	clearRegistries()
	
	// Parser that accepts URIs starting with "valid://"
	validParser := &mockURIParser{
		driverType: "valid",
		supportedSchemes: []string{"valid"},
		parseFunc: func(uri string) (types.Config, error) {
			if len(uri) > 8 && uri[:8] == "valid://" {
				return types.Config{
					Type: "valid",
					Host: uri[8:],
				}, nil
			}
			return types.Config{}, fmt.Errorf("not a valid URI")
		},
	}
	
	// Parser that accepts URIs starting with "test://"
	testParser := &mockURIParser{
		driverType: "test",
		supportedSchemes: []string{"test"},
		parseFunc: func(uri string) (types.Config, error) {
			if len(uri) > 7 && uri[:7] == "test://" {
				return types.Config{
					Type: "test",
					Host: uri[7:],
				}, nil
			}
			return types.Config{}, fmt.Errorf("not a test URI")
		},
	}
	
	RegisterURIParser("valid", validParser)
	RegisterURIParser("test", testParser)
	
	tests := []struct {
		name     string
		uri      string
		wantType string
		wantHost string
		wantErr  bool
	}{
		{
			name:     "valid URI for first parser",
			uri:      "valid://localhost:5432",
			wantType: "valid",
			wantHost: "localhost:5432",
			wantErr:  false,
		},
		{
			name:     "valid URI for second parser",
			uri:      "test://example.com",
			wantType: "test",
			wantHost: "example.com",
			wantErr:  false,
		},
		{
			name:    "invalid URI for all parsers",
			uri:     "invalid://something",
			wantErr: true,
		},
		{
			name:    "empty URI",
			uri:     "",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseURI(tt.uri)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseURI() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if tt.wantErr {
				return
			}
			
			if config.Type != tt.wantType {
				t.Errorf("ParseURI() Type = %v, want %v", config.Type, tt.wantType)
			}
			
			if config.Host != tt.wantHost {
				t.Errorf("ParseURI() Host = %v, want %v", config.Host, tt.wantHost)
			}
		})
	}
}

func TestParseURINoParserRegistered(t *testing.T) {
	clearRegistries()
	
	// No parsers registered
	_, err := ParseURI("test://something")
	if err == nil {
		t.Error("ParseURI() should return error when no parsers registered")
	}
	if err.Error() != "no URI parsers registered" {
		t.Errorf("ParseURI() error = %v, want 'no URI parsers registered'", err)
	}
}

func TestConcurrentAccess(t *testing.T) {
	clearRegistries()
	
	// Test concurrent driver registration and retrieval
	var wg sync.WaitGroup
	numGoroutines := 10
	
	// Start multiple goroutines registering different drivers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			driverType := fmt.Sprintf("driver%d", id)
			factory := func(config types.Config) (types.Database, error) {
				return nil, nil
			}
			Register(driverType, factory)
		}(i)
	}
	
	// Start multiple goroutines reading drivers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			driverType := fmt.Sprintf("driver%d", id)
			// Try to get a driver that may or may not be registered yet
			Get(driverType)
		}(i)
	}
	
	// Wait for all goroutines to complete
	wg.Wait()
	
	// Verify all drivers were registered
	for i := 0; i < numGoroutines; i++ {
		driverType := fmt.Sprintf("driver%d", i)
		_, err := Get(driverType)
		if err != nil {
			t.Errorf("Driver %s not found after concurrent registration", driverType)
		}
	}
}