package main

import (
	"flag"
	"reflect"
	"testing"
)

func TestInterspersedFlagArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "flag before path",
			args: []string{"-c", "host:/project"},
			want: []string{"-c", "host:/project"},
		},
		{
			name: "flag after path",
			args: []string{"host:/project", "-c"},
			want: []string{"-c", "host:/project"},
		},
		{
			name: "value flag after path",
			args: []string{"host:/project", "-log", "veda.log", "-c"},
			want: []string{"-log", "veda.log", "-c", "host:/project"},
		},
		{
			name: "terminator preserves trailing arguments",
			args: []string{"host:/project", "--", "-c"},
			want: []string{"--", "host:/project", "-c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.Bool("c", false, "")
			fs.String("log", "", "")
			got := interspersedFlagArgs(fs, tt.args)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("interspersedFlagArgs() = %q, want %q", got, tt.want)
			}
		})
	}
}
