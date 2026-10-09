package runner

import (
	"reflect"
	"testing"
)

func TestSplit(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"adb -s ABC shell id", []string{"adb", "-s", "ABC", "shell", "id"}},
		{"echo 'hello world'", []string{"echo", "hello world"}},
		{`echo "hello world"`, []string{"echo", "hello world"}},
		{`echo a\ b`, []string{"echo", "a b"}},
		{"  spaced   out  ", []string{"spaced", "out"}},
		{"single", []string{"single"}},
		{"", nil},
		{"   ", nil},
		{`mixed 'single' "double"`, []string{"mixed", "single", "double"}},
	}
	for _, c := range cases {
		got, err := Split(c.in)
		if err != nil {
			t.Errorf("Split(%q) returned error %v", c.in, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Split(%q) = %#v, want %#v", c.in, got, c.want)
		}
	}
}

func TestSplitRejectsUnbalancedQuote(t *testing.T) {
	if _, err := Split(`echo "unterminated`); err == nil {
		t.Error("expected an error for an unbalanced quote")
	}
}
