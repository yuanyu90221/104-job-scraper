package main

import "testing"

func TestNewRootCmd_LineFlagsDefaultFromEnv(t *testing.T) {
	t.Setenv("LINE_CHANNEL_SECRET", "env-secret")
	t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "env-token")
	t.Setenv("LINE_TARGET_ID", "env-target")

	cmd := newRootCmd()

	tests := []struct {
		flag string
		want string
	}{
		{"line-channel-secret", "env-secret"},
		{"line-channel-token", "env-token"},
		{"line-target-id", "env-target"},
	}
	for _, tt := range tests {
		f := cmd.Flags().Lookup(tt.flag)
		if f == nil {
			t.Fatalf("flag %q not registered", tt.flag)
		}
		if f.DefValue != tt.want {
			t.Errorf("flag %q default = %q, want %q", tt.flag, f.DefValue, tt.want)
		}
	}
}

func TestNewRootCmd_LineFlagsDefaultEmptyWithoutEnv(t *testing.T) {
	t.Setenv("LINE_CHANNEL_SECRET", "")
	t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "")
	t.Setenv("LINE_TARGET_ID", "")

	cmd := newRootCmd()

	for _, flag := range []string{"line-channel-secret", "line-channel-token", "line-target-id"} {
		f := cmd.Flags().Lookup(flag)
		if f.DefValue != "" {
			t.Errorf("flag %q default = %q, want empty", flag, f.DefValue)
		}
	}
}

func TestNewRootCmd_LineFlagExplicitlySetOverridesEnv(t *testing.T) {
	t.Setenv("LINE_CHANNEL_SECRET", "env-secret")

	cmd := newRootCmd()
	if err := cmd.Flags().Set("line-channel-secret", "flag-secret"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := cmd.Flags().GetString("line-channel-secret")
	if err != nil {
		t.Fatalf("GetString: %v", err)
	}
	if got != "flag-secret" {
		t.Errorf("line-channel-secret = %q, want %q", got, "flag-secret")
	}
}
