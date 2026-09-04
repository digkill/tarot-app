package decks

import (
	"encoding/json"
	"regexp"
	"strings"
)

var hexColor = regexp.MustCompile(`(?i)^#([0-9a-f]{6}|[0-9a-f]{3})$`)

type Theme struct {
	Bg     string `json:"bg"`
	Panel  string `json:"panel"`
	Accent string `json:"accent"`
	Text   string `json:"text"`
	Muted  string `json:"muted"`
	Gold   string `json:"gold"`
	Danger string `json:"danger"`
	TabBar string `json:"tabBar"`
}

var ClassicTheme = Theme{
	Bg:     "#040307",
	Panel:  "#1a1030",
	Accent: "#6c5ce7",
	Text:   "#f7f4ea",
	Muted:  "#9a93b3",
	Gold:   "#d4af37",
	Danger: "#ff6b6b",
	TabBar: "#0c0a14",
}

var Presets = map[string]Theme{
	"classic": ClassicTheme,
	"japanese": {
		Bg:     "#12090a",
		Panel:  "#2a1416",
		Accent: "#c44536",
		Text:   "#f4ead8",
		Muted:  "#c4a99a",
		Gold:   "#d4a017",
		Danger: "#ff6b6b",
		TabBar: "#0c0606",
	},
	"ink": {
		Bg:     "#07080c",
		Panel:  "#161b28",
		Accent: "#4c8dff",
		Text:   "#eef3ff",
		Muted:  "#8fa0c0",
		Gold:   "#c9d4e8",
		Danger: "#ff6b6b",
		TabBar: "#05060a",
	},
	"forest": {
		Bg:     "#071109",
		Panel:  "#13241a",
		Accent: "#2f9e6b",
		Text:   "#eef6ea",
		Muted:  "#9bb5a4",
		Gold:   "#c6a35a",
		Danger: "#ff6b6b",
		TabBar: "#050c07",
	},
}

func NormalizeTheme(t Theme) Theme {
	out := ClassicTheme
	set := func(dst *string, v string) {
		v = strings.TrimSpace(v)
		if hexColor.MatchString(v) {
			*dst = strings.ToLower(v)
		}
	}
	set(&out.Bg, t.Bg)
	set(&out.Panel, t.Panel)
	set(&out.Accent, t.Accent)
	set(&out.Text, t.Text)
	set(&out.Muted, t.Muted)
	set(&out.Gold, t.Gold)
	set(&out.Danger, t.Danger)
	set(&out.TabBar, t.TabBar)
	return out
}

func ThemeFromJSON(raw []byte) Theme {
	if len(raw) == 0 {
		return ClassicTheme
	}
	var t Theme
	if err := json.Unmarshal(raw, &t); err != nil {
		return ClassicTheme
	}
	return NormalizeTheme(t)
}

func (t Theme) JSON() []byte {
	t = NormalizeTheme(t)
	b, _ := json.Marshal(t)
	return b
}
