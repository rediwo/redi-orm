package database

import (
	"testing"
	"github.com/rediwo/redi-orm/types"
)

func TestParseURI(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		want    types.Config
		wantErr bool
	}{
		{
			name: "SQLite file path",
			uri:  "sqlite:///path/to/database.db",
			want: types.Config{
				Type:     types.SQLite,
				FilePath: "/path/to/database.db",
			},
		},
		{
			name: "SQLite memory",
			uri:  "sqlite://:memory:",
			want: types.Config{
				Type:     types.SQLite,
				FilePath: ":memory:",
			},
		},
		{
			name: "MySQL full URI",
			uri:  "mysql://user:pass@localhost:3306/testdb",
			want: types.Config{
				Type:     types.MySQL,
				Host:     "localhost",
				Port:     3306,
				Database: "testdb",
				User:     "user",
				Password: "pass",
			},
		},
		{
			name: "MySQL default port",
			uri:  "mysql://user:pass@localhost/testdb",
			want: types.Config{
				Type:     types.MySQL,
				Host:     "localhost",
				Port:     3306,
				Database: "testdb",
				User:     "user",
				Password: "pass",
			},
		},
		{
			name: "PostgreSQL full URI",
			uri:  "postgresql://user:pass@localhost:5432/testdb",
			want: types.Config{
				Type:     types.PostgreSQL,
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "user",
				Password: "pass",
			},
		},
		{
			name: "PostgreSQL alias",
			uri:  "postgres://user:pass@localhost:5432/testdb",
			want: types.Config{
				Type:     types.PostgreSQL,
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "user",
				Password: "pass",
			},
		},
		{
			name:    "Invalid scheme",
			uri:     "mongodb://localhost/test",
			wantErr: true,
		},
		{
			name:    "Invalid URI",
			uri:     "not a uri",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseURI(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseURI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseURI() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestNewFromURI(t *testing.T) {
	// Test SQLite memory database
	db, err := NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create SQLite database from URI: %v", err)
	}
	defer db.Close()

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to SQLite: %v", err)
	}
}