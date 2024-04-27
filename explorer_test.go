package main

import (
	"fmt"
	"testing"
)

func Test_fullformInputFieldPath(t *testing.T) {
	tests := []struct {
		inputFieldPath string
		fullformedKind string
		want           string
	}{
		{
			inputFieldPath: "sts.spec",
			fullformedKind: "statefulset",
			want:           "statefulset.spec",
		},
		{
			inputFieldPath: "sts",
			fullformedKind: "statefulset",
			want:           "statefulset",
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("make %s full-formed", tt.inputFieldPath), func(t *testing.T) {
			if got := fullformInputFieldPath(tt.inputFieldPath, tt.fullformedKind); got != tt.want {
				t.Errorf("fullformInputFieldPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
