package theme

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/zshimonz/lmdb-gui-client/bundle"
)

type MyDarkTheme struct{}

func (MyDarkTheme) Color(c fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	// always use dark theme
	return theme.DarkTheme().Color(c, v)
}

func (MyDarkTheme) Font(s fyne.TextStyle) fyne.Resource {
	if s.Monospace {
		return bundle.ResourceHackSaraRegularTtf
	}
	if s.Bold {
		if s.Italic {
			return theme.DefaultTheme().Font(s)
		}
		return theme.DefaultTheme().Font(s)
	}
	if s.Italic {
		return theme.DefaultTheme().Font(s)
	}
	return bundle.ResourceHackSaraRegularTtf
}

func (MyDarkTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

func (MyDarkTheme) Size(s fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(s)
}

type MyLightTheme struct{}

func (MyLightTheme) Color(c fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	// always use dark theme
	return theme.LightTheme().Color(c, v)
}

func (MyLightTheme) Font(s fyne.TextStyle) fyne.Resource {
	if s.Monospace {
		return bundle.ResourceHackSaraRegularTtf
	}
	if s.Bold {
		if s.Italic {
			return theme.DefaultTheme().Font(s)
		}
		return theme.DefaultTheme().Font(s)
	}
	if s.Italic {
		return theme.DefaultTheme().Font(s)
	}
	return bundle.ResourceHackSaraRegularTtf
}

func (MyLightTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

func (MyLightTheme) Size(s fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(s)
}
