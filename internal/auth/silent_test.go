package auth

import "testing"

func TestClassifySilentState(t *testing.T) {
	tests := []struct {
		state string
		want  silentAction
	}{
		{"done", silentDone},
		{"aad_tile", silentClickTile},
		{"kmsi", silentWait},
		{"next_button", silentWait},
		{"waiting:ConvergedSignIn:Sign in:", silentWait},
		{"waiting:::", silentWait},
		{"email_page", silentInteractive},
		{"switch_to_pwd", silentInteractive},
		{"pwd_page", silentInteractive},
		{"pick_method", silentInteractive},
		{"sms_tile", silentInteractive},
		{"sms_code", silentInteractive},
		{"error:Your account or password is incorrect", silentInteractive},
		{"", silentInteractive},
		{"something_new", silentInteractive},
	}
	for _, tt := range tests {
		if got := classifySilentState(tt.state); got != tt.want {
			t.Errorf("classifySilentState(%q) = %d, want %d", tt.state, got, tt.want)
		}
	}
}
