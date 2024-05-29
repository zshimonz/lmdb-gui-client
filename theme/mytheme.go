package theme

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"image/color"

	"github.com/zshimonz/lmdb-gui-client/data"
)

// MyTheme 定义自定义主题以使用嵌入的字体
type MyTheme struct{}

func (m *MyTheme) Font(s fyne.TextStyle) fyne.Resource {
	// load the font from the embedded resources
	return data.ResourceHackSaraRegularTtf
}

func (m *MyTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(name, variant)
}

func (m *MyTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m *MyTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
