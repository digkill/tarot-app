package decks

import "testing"

func TestMapFilenameHQ(t *testing.T) {
	cases := map[string]string{
		"00_THE_FOOL.png":         "the_fool",
		"08_STRENGTH.png":         "the_strength",
		"10_WHEEL_OF_FORTUNE.png": "wheel_of_fortune",
		"20_JUDGEMENT.jpg":        "judgement",
		"ACE_OF_WANDS.png":        "ace_of_wands",
		"CARD_BACK.png":           FileBack,
		"the_fool.jpeg":           "the_fool",
		"COVER.PNG":               FileCover,
		"12_THE_HANGED_MAN.png":   "the_hanged_man",
	}
	for in, want := range cases {
		got, ok := MapFilename(in)
		if !ok || got != want {
			t.Errorf("MapFilename(%q)=%q,%v want %q", in, got, ok, want)
		}
	}
	if _, ok := MapFilename("readme.txt"); ok {
		t.Fatal("readme should be skipped")
	}
}
