package helper

import (
	"testing"

	"one-click-aks-server/internal/entity"
)

func TestConvertStructToEnvVars(t *testing.T) {
	// Test with Preference struct
	preference := entity.Preference{
		AzureRegion:        "East US",
		UserDefaultVMSize:  "Standard_DS2_v2",
		TerminalAutoScroll: true,
	}

	envVars := ConvertStructToEnvVars(preference, "USER_PREF_")

	tests := []struct {
		key      string
		expected string
	}{
		{"USER_PREF_AZURE_REGION", "East US"},
		{"USER_PREF_USER_DEFAULT_VM_SIZE", "Standard_DS2_v2"},
		{"USER_PREF_TERMINAL_AUTO_SCROLL", "true"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if val, ok := envVars[tt.key]; !ok {
				t.Errorf("Expected key %s not found in envVars", tt.key)
			} else if val != tt.expected {
				t.Errorf("For key %s, expected value %s, got %s", tt.key, tt.expected, val)
			}
		})
	}

	// Verify we have exactly 3 environment variables
	if len(envVars) != 3 {
		t.Errorf("Expected 3 environment variables, got %d", len(envVars))
	}
}

func TestCamelToConventional(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"AzureRegion", "azure_region"},
		{"UserDefaultVMSize", "user_default_vm_size"},
		{"TerminalAutoScroll", "terminal_auto_scroll"},
		{"HTTPServer", "http_server"},
		{"simpleCase", "simple_case"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := CamelToConventional(tt.input)
			if result != tt.expected {
				t.Errorf("For input %s, expected %s, got %s", tt.input, tt.expected, result)
			}
		})
	}
}
