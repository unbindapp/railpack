package plan

import (
	"encoding/json"
	"testing"
)

func TestCommandMarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		command  Command
		expected string
	}{
		{
			name:     "exec command without custom name",
			command:  NewExecCommand("echo hello"),
			expected: `"RUN:echo hello"`,
		},
		{
			name:     "exec command with custom name",
			command:  NewExecCommand("echo hello", "Say Hello"),
			expected: `"RUN#Say Hello:echo hello"`,
		},
		{
			name:     "path command without custom name",
			command:  NewPathCommand("/usr/local/bin"),
			expected: `"PATH:/usr/local/bin"`,
		},
		{
			name:     "variable command without custom name",
			command:  NewVariableCommand("KEY", "value"),
			expected: `"ENV:KEY=value"`,
		},
		{
			name:     "copy command without custom name",
			command:  NewCopyCommand("src.txt", "dst.txt"),
			expected: `"COPY:src.txt dst.txt"`,
		},
		{
			name:     "file command without custom name",
			command:  NewFileCommand("/etc/conf", "config.yaml"),
			expected: `"FILE:/etc/conf config.yaml"`,
		},
		{
			name:     "file command with custom name",
			command:  NewFileCommand("/etc/conf", "config.yaml", "Config File"),
			expected: `"FILE#Config File:/etc/conf config.yaml"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshalling
			data, err := json.Marshal(tt.command)
			if err != nil {
				t.Fatalf("failed to marshal command: %v", err)
			}
			if string(data) != tt.expected {
				t.Errorf("marshal result\ngot:  %s\nwant: %s", string(data), tt.expected)
			}

			// Test unmarshalling
			cmd, err := UnmarshalCommand(data)
			if err != nil {
				t.Fatalf("failed to unmarshal command: %v", err)
			}

			// Marshal again to verify it produces the same result
			roundTrip, err := json.Marshal(cmd)
			if err != nil {
				t.Fatalf("failed to marshal unmarshalled command: %v", err)
			}
			if string(roundTrip) != tt.expected {
				t.Errorf("round-trip result\ngot:  %s\nwant: %s", string(roundTrip), tt.expected)
			}
		})
	}
}
