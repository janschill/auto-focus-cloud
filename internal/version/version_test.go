package version

import (
	"testing"
)

func TestIsCompatible_SameMajorVersion(t *testing.T) {
	tests := []struct {
		name             string
		licenseVersion   string
		requestedVersion string
		expected         bool
		expectError      bool
	}{
		{
			name:             "same version",
			licenseVersion:   "1.0.0",
			requestedVersion: "1.0.0",
			expected:         true,
			expectError:      false,
		},
		{
			name:             "same major, different minor",
			licenseVersion:   "1.0.0",
			requestedVersion: "1.5.0",
			expected:         true,
			expectError:      false,
		},
		{
			name:             "same major, different patch",
			licenseVersion:   "1.0.0",
			requestedVersion: "1.0.5",
			expected:         true,
			expectError:      false,
		},
		{
			name:             "different major version",
			licenseVersion:   "1.0.0",
			requestedVersion: "2.0.0",
			expected:         false,
			expectError:      false,
		},
		{
			name:             "different major version reverse",
			licenseVersion:   "2.0.0",
			requestedVersion: "1.0.0",
			expected:         false,
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := IsCompatible(tt.licenseVersion, tt.requestedVersion)
			
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			if result != tt.expected {
				t.Errorf("IsCompatible(%q, %q) = %v, want %v", 
					tt.licenseVersion, tt.requestedVersion, result, tt.expected)
			}
		})
	}
}

func TestIsCompatible_ErrorCases(t *testing.T) {
	tests := []struct {
		name             string
		licenseVersion   string
		requestedVersion string
		expectError      bool
	}{
		{
			name:             "empty license version",
			licenseVersion:   "",
			requestedVersion: "1.0.0",
			expectError:      true,
		},
		{
			name:             "empty requested version",
			licenseVersion:   "1.0.0",
			requestedVersion: "",
			expectError:      true,
		},
		{
			name:             "invalid license version",
			licenseVersion:   "invalid",
			requestedVersion: "1.0.0",
			expectError:      true,
		},
		{
			name:             "invalid requested version",
			licenseVersion:   "1.0.0",
			requestedVersion: "invalid",
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := IsCompatible(tt.licenseVersion, tt.requestedVersion)
			
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestExtractMajorVersion_ValidVersions(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected int
	}{
		{"single digit", "1.0.0", 1},
		{"double digit", "12.5.3", 12},
		{"zero major", "0.1.0", 0},
		{"large major", "999.0.0", 999},
		{"only major", "5", 5},
		{"major.minor", "3.2", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractMajorVersion(tt.version)
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			if result != tt.expected {
				t.Errorf("ExtractMajorVersion(%q) = %d, want %d", tt.version, result, tt.expected)
			}
		})
	}
}

func TestExtractMajorVersion_InvalidVersions(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{"empty string", ""},
		{"non-numeric major", "v1.0.0"},
		{"negative major", "-1.0.0"},
		{"alpha major", "a.0.0"},
		{"space in version", " 1.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExtractMajorVersion(tt.version)
			
			if err == nil {
				t.Errorf("Expected error for version %q, but got none", tt.version)
			}
		})
	}
}

func TestExtractMajorVersion_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		expectError bool
		expected    int
	}{
		{"version with build metadata", "1.0.0+build.123", false, 1},
		{"version with pre-release", "2.0.0-alpha.1", false, 2},
		{"version with leading zeros", "01.0.0", false, 1},
		{"very long version", "123.456.789.012.345", false, 123},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractMajorVersion(tt.version)
			
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			if !tt.expectError && result != tt.expected {
				t.Errorf("ExtractMajorVersion(%q) = %d, want %d", tt.version, result, tt.expected)
			}
		})
	}
}

// Benchmark version functions
func BenchmarkIsCompatible(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = IsCompatible("1.0.0", "1.5.0")
	}
}

func BenchmarkExtractMajorVersion(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ExtractMajorVersion("1.2.3")
	}
}