package theme

import "veda/internal/ui"

// FunctionHeaderBackground blends a subtle blue tint into the theme background.
func (t *Theme) FunctionHeaderBackground() ui.Color {
	normal := t.GetStyle(t.Normal).Bg
	if normal == ui.ColorDefault {
		return normal
	}
	r, g, b := normal.RGB()
	return ui.NewRGBColor((7*r+70)/8, (7*g+125)/8, (7*b+220)/8)
}
