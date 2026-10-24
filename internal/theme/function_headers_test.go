package theme

import (
	"testing"
	"veda/internal/ui"
)

func TestFunctionHeaderBackground(t *testing.T) {
	for _, name := range GetThemeNames() {
		t.Run(name, func(t *testing.T) {
			theme := GetThemeByName(name)
			if got := theme.FunctionHeaderBackground(); got == theme.GetStyle(theme.Normal).Bg {
				t.Fatal("function header must be distinct from normal background")
			}
		})
	}
	custom := &Theme{}
	if custom.FunctionHeaderBackground() != ui.ColorDefault {
		t.Fatal("unspecified colors must retain terminal background")
	}
}

func TestFunctionHeaderBackgroundIsBlue(t *testing.T) {
	custom := &Theme{
		Normal:         StyleDef{Background: "#202020"},
		SyntaxFunction: StyleDef{Foreground: "#00ff00"},
	}
	r, g, b := custom.FunctionHeaderBackground().RGB()
	if b <= g || g <= r {
		t.Fatalf("header tint must be blue regardless of function color; got (%d,%d,%d)", r, g, b)
	}
}
