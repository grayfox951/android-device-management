package i18n

import "testing"

// TestNoMissingKeys guards against a translation silently falling back to
// English: every locale must cover the whole English key set.
func TestNoMissingKeys(t *testing.T) {
	for _, l := range Order {
		t.Run(string(l), func(t *testing.T) {
			missing := Missing(l)
			if len(missing) == 0 {
				return
			}
			for _, k := range missing {
				t.Errorf("missing translation for %q", k)
			}
		})
	}
}

// TestNoEmptyTranslations catches keys present but blank, which would render as
// an empty label instead of falling back.
func TestNoEmptyTranslations(t *testing.T) {
	for _, l := range Order {
		for k, v := range dict[l] {
			if v == "" {
				t.Errorf("%s: key %q has an empty translation", l, k)
			}
		}
	}
}

// TestNoFormatMismatch verifies that the verbs in a translation match those in
// English, so fmt.Sprintf cannot produce %!(EXTRA …) noise in the interface.
func TestNoFormatMismatch(t *testing.T) {
	for _, l := range Order {
		for k, en := range dict[EN] {
			got, ok := dict[l][k]
			if !ok {
				continue
			}
			if verbCount(en) != verbCount(got) {
				t.Errorf("%s: key %q has %d format verbs, English has %d: %q vs %q",
					l, k, verbCount(got), verbCount(en), got, en)
			}
		}
	}
}

// verbCount counts the % directives that consume an argument.
func verbCount(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] != '%' {
			continue
		}
		// %% is a literal and consumes nothing.
		if i+1 < len(s) && s[i+1] == '%' {
			i++
			continue
		}
		n++
		for i+1 < len(s) && !isVerbEnd(s[i+1]) {
			i++
		}
	}
	return n
}

func isVerbEnd(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func TestGetFallsBackToEnglish(t *testing.T) {
	SetLanguage(DE)
	defer SetLanguage(RU)

	if got := Get("dev.pick"); got == "dev.pick" {
		t.Errorf("known key resolved to itself: %q", got)
	}
	if got := Get("no.such.key.at.all"); got != "no.such.key.at.all" {
		t.Errorf("unknown key should echo itself, got %q", got)
	}
}

func TestCurrentAndSet(t *testing.T) {
	for _, l := range Order {
		SetLanguage(l)
		if Current() != l {
			t.Fatalf("SetLanguage(%s) left Current() as %s", l, Current())
		}
	}
	// An unsupported code must fall back rather than break the interface.
	SetLanguage("fr")
	if Current() != EN {
		t.Errorf("unknown language fell back to %s, want en", Current())
	}
}
