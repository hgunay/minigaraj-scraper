package shared

import (
	"testing"
)

func TestExtractWikiPageName(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"normal wiki page", "https://hotwheels.fandom.com/wiki/Twin_Mill", "Twin Mill"},
		{"encoded characters", "https://hotwheels.fandom.com/wiki/%2768_Mustang", "'68 Mustang"},
		{"underscores to spaces", "https://hotwheels.fandom.com/wiki/Bone_Shaker_(2020)", "Bone Shaker (2020)"},
		{"matchbox wiki", "https://matchbox.fandom.com/wiki/Tesla_Model_S", "Tesla Model S"},
		{"no wiki path", "https://example.com/page", ""},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractWikiPageName(tt.url)
			if got != tt.want {
				t.Errorf("ExtractWikiPageName(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestCleanImageURL(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{"removes revision", "https://static.wikia.nocookie.net/img.jpg/revision/latest/scale-to-width-down/300", "https://static.wikia.nocookie.net/img.jpg"},
		{"no revision", "https://example.com/image.png", "https://example.com/image.png"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanImageURL(tt.src)
			if got != tt.want {
				t.Errorf("CleanImageURL(%q) = %q, want %q", tt.src, got, tt.want)
			}
		})
	}
}

func TestNormalizeScale(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"1:64", "1:64"},
		{"1/64", "1:64"},
		{" 1:64 ", "1:64"},
		{"1/43", "1:43"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeScale(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeScale(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s    string
		subs []string
		want bool
	}{
		{"wheel type", []string{"wheel"}, true},
		{"color", []string{"colour", "color"}, true},
		{"name", []string{"series", "year"}, false},
		{"", []string{"x"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := Contains(tt.s, tt.subs...)
			if got != tt.want {
				t.Errorf("Contains(%q, %v) = %v, want %v", tt.s, tt.subs, got, tt.want)
			}
		})
	}
}

func TestStrPtr(t *testing.T) {
	if got := StrPtr("hello"); got == nil || *got != "hello" {
		t.Error("StrPtr(\"hello\") should return pointer to \"hello\"")
	}
	if got := StrPtr(""); got != nil {
		t.Error("StrPtr(\"\") should return nil")
	}
}

func TestYearRegex(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"List of 2024 Hot Wheels", "2024"},
		{"no year here", ""},
		{"model from 1968", "1968"},
		{"year 2099 invalid", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			matches := YearRegex.FindStringSubmatch(tt.input)
			got := ""
			if len(matches) > 1 {
				got = matches[1]
			}
			if got != tt.want {
				t.Errorf("YearRegex on %q = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRefNumRegex(t *testing.T) {
	tests := []struct {
		input string
		match bool
	}{
		{"GJR456", true},
		{"HW1234", true},
		{"ABC12345Z", true},
		{"lowercase", false},
		{"AB", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := RefNumRegex.MatchString(tt.input)
			if got != tt.match {
				t.Errorf("RefNumRegex.MatchString(%q) = %v, want %v", tt.input, got, tt.match)
			}
		})
	}
}
