package i18n

import (
	"fmt"
	"sort"
)

// Lang is a supported interface language.
type Lang string

const (
	EN Lang = "en"
	RU Lang = "ru"
	UK Lang = "uk"
	BE Lang = "be"
	DE Lang = "de"
)

// Names holds the autonym of each language, used in the language picker.
var Names = map[Lang]string{
	EN: "English",
	RU: "Русский",
	UK: "Українська",
	BE: "Беларуская",
	DE: "Deutsch",
}

// Order is the order the languages are offered in.
var Order = []Lang{EN, RU, UK, BE, DE}

var dict = map[Lang]map[string]string{
}

// missingKeys collects keys present in EN but absent in a translation, so the
// console can warn about gaps during development instead of silently showing
// English text.
func missingKeys(l Lang) []string {
	var out []string
	for k := range dict[EN] {
		if _, ok := dict[l][k]; !ok {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// Missing reports untranslated keys for the given language.
func Missing(l Lang) []string { return missingKeys(l) }

// SetLanguage selects the active language. Unknown languages fall back to EN.
func SetLanguage(l Lang) {
	if _, ok := dict[l]; !ok {
		l = EN
	}
	cur = l
}

var cur = RU

// Get returns the translation of key in the active language.
func Get(key string) string {
	if m, ok := dict[cur]; ok {
		if s, ok := m[key]; ok {
			return s
		}
	}
	if s, ok := dict[EN][key]; ok {
		return s
	}
	return key
}

// Current returns the active language.
func Current() Lang { return cur }

// Has reports whether key is defined in English. It lets callers assert that
// every key they reference actually exists, instead of silently rendering the
// key itself when it does not.
func Has(key string) bool {
	_, ok := dict[EN][key]
	return ok
}

// Keys returns every key defined in English.
func Keys() []string {
	out := make([]string, 0, len(dict[EN]))
	for k := range dict[EN] {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// T is Get with formatting.
func T(key string, args ...any) string {
	return fmt.Sprintf(Get(key), args...)
}
