package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"hillside/internal/utils"

	"github.com/gdamore/tcell/v2"
	"gopkg.in/yaml.v3"
)

// ThemeConfig represents a theme loaded from YAML
type ThemeConfig struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Author      string         `yaml:"author"`
	Version     string         `yaml:"version"`
	Colors      map[string]any `yaml:"colors"`
}

// Theme represents a processed theme with tcell colors
type Theme struct {
	Name        string
	Description string
	Author      string
	Version     string
	colors      map[string]tcell.Color
}

// LoadTheme loads a theme from a YAML file
func LoadTheme(themePath string) (*Theme, error) {
	data, err := os.ReadFile(themePath)
	if err != nil {
		return nil, utils.ThemeError(fmt.Sprintf("failed to read theme file: %v", err))
	}

	var config ThemeConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, utils.ThemeError(fmt.Sprintf("failed to parse theme YAML: %v", err))
	}

	theme := &Theme{
		Name:        config.Name,
		Description: config.Description,
		Author:      config.Author,
		Version:     config.Version,
		colors:      make(map[string]tcell.Color),
	}

	// Process colors
	for key, value := range config.Colors {
		color, err := parseColor(value)
		if err != nil {
			return nil, utils.ThemeError(fmt.Sprintf("failed to parse color '%s': %v", key, err))
		}
		theme.colors[key] = color
	}

	return theme, nil
}

func LoadThemeFromDir(themesDir, themeName string) (*Theme, error) {
	themePath := filepath.Join(themesDir, themeName+".yaml")
	theme, err := LoadTheme(themePath)
	if err != nil {
		// Try loading from built-in themes
		themesDir := filepath.Join("..", "..", "themes")
		files, err := os.ReadDir(themesDir)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".yaml") {
				themePath := filepath.Join(themesDir, file.Name())
				return LoadTheme(themePath)
			}
		}
		return nil, fmt.Errorf("no .yaml theme found in %s", themesDir)
	}
	return theme, nil
}

// GetColor returns a color by name, with fallback to default
func (t *Theme) GetColor(name string) tcell.Color {
	if color, exists := t.colors[name]; exists {
		return color
	}
	// Fallback to white if color not found
	return tcell.ColorWhite
}

// GetColorWithFallback returns a color by name with a custom fallback
func (t *Theme) GetColorWithFallback(name string, fallback tcell.Color) tcell.Color {
	if color, exists := t.colors[name]; exists {
		return color
	}
	return fallback
}

// HasColor checks if a color exists in the theme
func (t *Theme) HasColor(name string) bool {
	_, exists := t.colors[name]
	return exists
}

// ListColors returns all available color names
func (t *Theme) ListColors() []string {
	keys := make([]string, 0, len(t.colors))
	for key := range t.colors {
		keys = append(keys, key)
	}
	return keys
}

// parseColor converts various color formats to tcell.Color
func parseColor(value any) (tcell.Color, error) {
	switch v := value.(type) {
	case string:
		return parseColorString(v)
	case int:
		return tcell.Color(v), nil
	case map[string]any:
		return parseColorMap(v)
	default:
		return tcell.ColorWhite, utils.ThemeError(fmt.Sprintf("unsupported color format: %T", value))
	}
}

// parseColorString parses color strings in various formats
func parseColorString(colorStr string) (tcell.Color, error) {
	colorStr = strings.TrimSpace(colorStr)

	// Handle hex colors
	if strings.HasPrefix(colorStr, "#") {
		return parseHexColor(colorStr)
	}

	// Handle RGB function format: rgb(255, 255, 255)
	if strings.HasPrefix(colorStr, "rgb(") && strings.HasSuffix(colorStr, ")") {
		return parseRGBFunction(colorStr)
	}

	// Handle named colors
	return parseNamedColor(colorStr)
}

// parseHexColor parses hex color strings like #FF0000 or #f00
func parseHexColor(hex string) (tcell.Color, error) {
	hex = strings.TrimPrefix(hex, "#")

	var r, g, b int64
	var err error

	switch len(hex) {
	case 3: // #RGB -> #RRGGBB
		r, err = strconv.ParseInt(string(hex[0])+string(hex[0]), 16, 64)
		if err != nil {
			return tcell.ColorWhite, err
		}
		g, err = strconv.ParseInt(string(hex[1])+string(hex[1]), 16, 64)
		if err != nil {
			return tcell.ColorWhite, err
		}
		b, err = strconv.ParseInt(string(hex[2])+string(hex[2]), 16, 64)
		if err != nil {
			return tcell.ColorWhite, err
		}
	case 6: // #RRGGBB
		r, err = strconv.ParseInt(hex[0:2], 16, 64)
		if err != nil {
			return tcell.ColorWhite, err
		}
		g, err = strconv.ParseInt(hex[2:4], 16, 64)
		if err != nil {
			return tcell.ColorWhite, err
		}
		b, err = strconv.ParseInt(hex[4:6], 16, 64)
		if err != nil {
			return tcell.ColorWhite, err
		}
	default:
		return tcell.ColorWhite, utils.ThemeError(fmt.Sprintf("invalid hex color format: %s", hex))
	}

	return tcell.NewRGBColor(int32(r), int32(g), int32(b)), nil
}

// parseRGBFunction parses RGB function format: rgb(255, 255, 255)
func parseRGBFunction(rgbStr string) (tcell.Color, error) {
	rgbStr = strings.TrimPrefix(rgbStr, "rgb(")
	rgbStr = strings.TrimSuffix(rgbStr, ")")

	parts := strings.Split(rgbStr, ",")
	if len(parts) != 3 {
		return tcell.ColorWhite, fmt.Errorf("invalid RGB format: %s", rgbStr)
	}

	r, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return tcell.ColorWhite, err
	}
	g, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return tcell.ColorWhite, err
	}
	b, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil {
		return tcell.ColorWhite, err
	}

	return tcell.NewRGBColor(int32(r), int32(g), int32(b)), nil
}

// parseColorMap parses color maps with r, g, b values
func parseColorMap(colorMap map[string]any) (tcell.Color, error) {
	rVal, rExists := colorMap["r"]
	gVal, gExists := colorMap["g"]
	bVal, bExists := colorMap["b"]

	if !rExists || !gExists || !bExists {
		return tcell.ColorWhite, utils.ThemeError("RGB COLOR MAP MUST HAVE R, G, B VALUES")
	}

	r, ok := rVal.(int)
	if !ok {
		return tcell.ColorWhite, utils.ThemeError("RED VALUE MUST BE AN INTEGER")
	}
	g, ok := gVal.(int)
	if !ok {
		return tcell.ColorWhite, utils.ThemeError("GREEN VALUE MUST BE AN INTEGER")
	}
	b, ok := bVal.(int)
	if !ok {
		return tcell.ColorWhite, utils.ThemeError("BLUE VALUE MUST BE AN INTEGER")
	}

	return tcell.NewRGBColor(int32(r), int32(g), int32(b)), nil
}

// parseNamedColor converts named colors to tcell colors
func parseNamedColor(name string) (tcell.Color, error) {
	name = strings.ToLower(name)

	namedColors := map[string]tcell.Color{
		"black":   tcell.ColorBlack,
		"red":     tcell.ColorRed,
		"green":   tcell.ColorGreen,
		"yellow":  tcell.ColorYellow,
		"blue":    tcell.ColorBlue,
		"magenta": tcell.ColorDarkMagenta,
		"cyan":    tcell.ColorLightCyan,
		"white":   tcell.ColorWhite,
		"gray":    tcell.ColorGray,
		"grey":    tcell.ColorGray,
	}

	if color, exists := namedColors[name]; exists {
		return color, nil
	}

	return tcell.ColorWhite, utils.ThemeError(fmt.Sprintf("unknown color name: %s", name))
}

// Helper methods for common UI components
func (t *Theme) FormColors() (bg, fieldBg, buttonBg, buttonText, fieldText tcell.Color) {
	return t.GetColor("background"),
		t.GetColor("input-field"),
		t.GetColor("button-active"),
		t.GetColor("button-text"),
		t.GetColor("foreground")
}

func (t *Theme) TextViewColors() (bg, text tcell.Color) {
	return t.GetColor("background"), t.GetColor("foreground")
}

func (t *Theme) ModalColors() (bg, text, border tcell.Color) {
	return t.GetColor("modal-background"),
		t.GetColor("foreground"),
		t.GetColor("border")
}

func (t *Theme) BorderColors() (normal, focus tcell.Color) {
	return t.GetColor("border"), t.GetColor("border-focus")
}
