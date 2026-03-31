package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Palette — matches app.py color variables exactly.
var (
	colorBGPrimary   = hexColor(0x1a1a2e)
	colorBGSecondary = hexColor(0x16213e)
	colorBGSurface   = hexColor(0x0d2137)
	colorBGInput     = hexColor(0x0a1628)
	colorBorder      = hexColor(0x1e3a5f)
	colorAccent      = hexColor(0x4f8ef7)
	colorGreen       = hexColor(0x4caf50)
	colorYellow      = hexColor(0xffb300)
	colorRed         = hexColor(0xe94560)
	colorTextPrimary = hexColor(0xe8eaf6)
	colorTextMuted   = hexColor(0x7986a8)
	colorTextDim     = hexColor(0x4a5568)
)

func hexColor(rgb uint32) color.NRGBA {
	return color.NRGBA{
		R: uint8(rgb >> 16),
		G: uint8(rgb >> 8 & 0xff),
		B: uint8(rgb & 0xff),
		A: 0xff,
	}
}

// darkTheme is a custom Fyne theme matching the WSL Proxy dark palette.
type darkTheme struct{}

var _ fyne.Theme = (*darkTheme)(nil)

func (darkTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return colorBGPrimary
	case theme.ColorNameButton:
		return colorBGSecondary
	case theme.ColorNameDisabledButton:
		return colorBGSurface
	case theme.ColorNameDisabled:
		return colorTextDim
	case theme.ColorNameForeground:
		return colorTextPrimary
	case theme.ColorNameHover:
		return colorBGSurface
	case theme.ColorNameFocus:
		return colorAccent
	case theme.ColorNameInputBackground:
		return colorBGInput
	case theme.ColorNameInputBorder:
		return colorBorder
	case theme.ColorNamePlaceHolder:
		return colorTextDim
	case theme.ColorNamePressed:
		return colorBGInput
	case theme.ColorNamePrimary:
		return colorAccent
	case theme.ColorNameScrollBar:
		return colorBorder
	case theme.ColorNameSelection:
		return colorBGSurface
	case theme.ColorNameSeparator:
		return colorBorder
	case theme.ColorNameShadow:
		return color.NRGBA{0, 0, 0, 0x80}
	default:
		return theme.DefaultTheme().Color(name, theme.VariantDark)
	}
}

func (darkTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (darkTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (darkTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 13
	case theme.SizeNamePadding:
		return 6
	case theme.SizeNameInnerPadding:
		return 8
	case theme.SizeNameScrollBar:
		return 10
	case theme.SizeNameScrollBarSmall:
		return 4
	default:
		return theme.DefaultTheme().Size(name)
	}
}
