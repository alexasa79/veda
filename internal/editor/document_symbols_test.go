package editor

import (
	"encoding/json"
	"strings"
	"testing"
	"veda/internal/proxy"
)

func TestDocumentSymbolDisplayName(t *testing.T) {
	tests := []struct {
		name   string
		symbol string
		want   string
	}{
		{
			name:   "class method",
			symbol: `{"name":"refresh","containerName":"Panel"}`,
			want:   "Panel::refresh",
		},
		{
			name:   "same method in another class",
			symbol: `{"name":"refresh","containerName":"Dialog"}`,
			want:   "Dialog::refresh",
		},
		{
			name:   "class in namespace",
			symbol: `{"name":"refresh","containerName":"ui::Panel"}`,
			want:   "ui::Panel::refresh",
		},
		{
			name:   "already qualified",
			symbol: `{"name":"Panel::refresh","containerName":"Panel"}`,
			want:   "Panel::refresh",
		},
		{
			name:   "free function",
			symbol: `{"name":"refresh"}`,
			want:   "refresh",
		},
		{
			name:   "empty container",
			symbol: `{"name":"refresh","containerName":""}`,
			want:   "refresh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Decode the LSP fields so the test also catches dropped container names.
			var symbol proxy.DocumentSymbol
			if err := json.Unmarshal([]byte(tt.symbol), &symbol); err != nil {
				t.Fatal(err)
			}
			symbol.Kind = 6 // Method
			symbol.Range.Start.Line = 41
			// Ignore the icon, but verify the full label used for display and filtering.
			_, label, found := strings.Cut(documentSymbolDisplayName(symbol), " ")
			if want := tt.want + " (line 42)"; !found || label != want {
				t.Errorf("symbol label = %q, want %q", label, want)
			}
		})
	}
}
