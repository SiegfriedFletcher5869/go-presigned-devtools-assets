package main

import "testing"

func TestPrepareUploadDecision(t *testing.T) {
	cases := []struct {
		name    string
		event   BuildEvent
		wantErr bool
	}{{"valid", BuildEvent{"releases/bundle.js", "application/javascript", 42}, false}, {"missing key", BuildEvent{"", "text/plain", 4}, true}, {"zero bytes", BuildEvent{"a", "text/plain", 0}, true}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := validBuildEvent(tc.event); got == tc.wantErr {
				t.Fatalf("validBuildEvent()=%v, want %v", got, !tc.wantErr)
			}
		})
	}
}
