package service

import "testing"

func TestNormalizeInstagramHandle(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"@eight_two_five_":                            "eight_two_five_",
		"eight_two_five_":                             "eight_two_five_",
		"  @eight_two_five_  ":                        "eight_two_five_",
		"https://www.instagram.com/eight_two_five_/":  "eight_two_five_",
		"https://instagram.com/eight_two_five_?hl=en": "eight_two_five_",
		"instagram.com/eight_two_five_":               "eight_two_five_",
		"":                                            "",
		"   ":                                         "",
	}

	for input, want := range cases {
		if got := normalizeInstagramHandle(input); got != want {
			t.Errorf("normalizeInstagramHandle(%q) = %q, want %q", input, got, want)
		}
	}
}
