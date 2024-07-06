package v1

import (
	"reflect"
	"testing"
)

func TestFromString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  ModelVersion
		expectErr bool
	}{
		{
			name:      "Valid version 1.2.3",
			input:     "1.2.3",
			expected:  ModelVersion{Major: 1, Minor: 2, Patch: 3},
			expectErr: false,
		},
		{
			name:      "Invalid format missing part",
			input:     "1.2",
			expected:  ModelVersion{},
			expectErr: true,
		},
		{
			name:      "Invalid format extra part",
			input:     "1.2.3.4",
			expected:  ModelVersion{},
			expectErr: true,
		},
		{
			name:      "Non-numeric major version",
			input:     "a.2.3",
			expected:  ModelVersion{},
			expectErr: true,
		},
		{
			name:      "Non-numeric minor version",
			input:     "1.b.3",
			expected:  ModelVersion{},
			expectErr: true,
		},
		{
			name:      "Non-numeric patch version",
			input:     "1.2.c",
			expected:  ModelVersion{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ModelVersionFromString(tt.input)
			if (err != nil) != tt.expectErr {
				t.Errorf("ModelVersionFromString() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ModelVersionFromString() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
