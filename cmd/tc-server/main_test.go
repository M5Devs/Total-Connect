package main

import (
	"strings"
	"testing"
)

func TestIsLoopback(t *testing.T) {
	tests := []struct {
		host     string
		expected bool
	}{
		{"127.0.0.1", true},
		{"localhost", true},
		{"LocalHost", true},
		{"::1", true},
		{"127.0.0.2", true},
		{"0.0.0.0", false},
		{"192.168.1.100", false},
		{"10.0.0.1", false},
		{"example.com", false},
	}

	for _, tt := range tests {
		got := isLoopback(tt.host)
		if got != tt.expected {
			t.Errorf("isLoopback(%q) = %v; want %v", tt.host, got, tt.expected)
		}
	}
}

func TestValidateSecurity(t *testing.T) {
	tests := []struct {
		name           string
		host           string
		auth           string
		insecureNoAuth bool
		wantWarn       bool
		wantErr        bool
		errContains    string
	}{
		{
			name:           "loopback without auth",
			host:           "127.0.0.1",
			auth:           "",
			insecureNoAuth: false,
			wantWarn:       false,
			wantErr:        false,
		},
		{
			name:           "localhost without auth",
			host:           "localhost",
			auth:           "",
			insecureNoAuth: false,
			wantWarn:       false,
			wantErr:        false,
		},
		{
			name:           "non-loopback with auth",
			host:           "0.0.0.0",
			auth:           "admin:secret",
			insecureNoAuth: false,
			wantWarn:       false,
			wantErr:        false,
		},
		{
			name:           "non-loopback with whitespace auth",
			host:           "0.0.0.0",
			auth:           "   ",
			insecureNoAuth: false,
			wantWarn:       false,
			wantErr:        true,
			errContains:    "SECURITY ERROR: Refusing to bind to non-loopback interface",
		},
		{
			name:           "non-loopback without auth - refused",
			host:           "0.0.0.0",
			auth:           "",
			insecureNoAuth: false,
			wantWarn:       false,
			wantErr:        true,
			errContains:    "SECURITY ERROR: Refusing to bind to non-loopback interface (0.0.0.0) without authentication",
		},
		{
			name:           "non-loopback without auth - allowed with insecure flag",
			host:           "0.0.0.0",
			auth:           "",
			insecureNoAuth: true,
			wantWarn:       true,
			wantErr:        false,
		},
		{
			name:           "custom IP without auth - refused",
			host:           "192.168.1.50",
			auth:           "",
			insecureNoAuth: false,
			wantWarn:       false,
			wantErr:        true,
			errContains:    "SECURITY ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warn, err := validateSecurity(tt.host, tt.auth, tt.insecureNoAuth)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			if warn != tt.wantWarn {
				t.Errorf("warn = %v; want %v", warn, tt.wantWarn)
			}
		})
	}
}
