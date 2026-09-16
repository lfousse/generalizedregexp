package filter_test

import (
	"bufio"
	"bytes"
	"errors"
	"strings"
	"testing"
	"testing/iotest"

	gr "github.com/lfousse/generalizedregexp"
	"github.com/lfousse/generalizedregexp/filter"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		args []string
		// match and reject list lines the resulting expression must accept and refuse.
		match  []string
		reject []string
	}{
		{
			name:   "a single litteral",
			args:   []string{"foo"},
			match:  []string{"foo"},
			reject: []string{"", "fo", "foobar", "a foo b"},
		},
		{
			name:   "the empty litteral",
			args:   []string{""},
			match:  []string{""},
			reject: []string{"a"},
		},
		{
			// Only the first comma is removed, so ",,star" is the literal ",star".
			name:   "an escaped comma",
			args:   []string{",,star"},
			match:  []string{",star"},
			reject: []string{"", ",,star", "star"},
		},
		{
			name:   "an escaped comma, twice over",
			args:   []string{",,,star"},
			match:  []string{",,star"},
			reject: []string{",star"},
		},
		{
			// A comma only introduces an operator in leading position.
			name:   "a comma inside a litteral",
			args:   []string{"a,star"},
			match:  []string{"a,star"},
			reject: []string{"astar"},
		},
		{
			name:   "any",
			args:   []string{",any"},
			match:  []string{"", "a", "anything at all", "é"},
			reject: nil,
		},
		{
			name:   "empty",
			args:   []string{",empty"},
			match:  nil,
			reject: []string{"", "a", "anything at all"},
		},
		{
			name:   "dot",
			args:   []string{",dot"},
			match:  []string{"a", "é", "🙂"},
			reject: []string{"", "ab"},
		},
		{
			// Operands are consumed in the order they were pushed.
			name:   "concat",
			args:   []string{"foo", "bar", ",concat"},
			match:  []string{"foobar"},
			reject: []string{"barfoo", "foo", "bar", ""},
		},
		{
			name:   "or",
			args:   []string{"foo", "bar", ",or"},
			match:  []string{"foo", "bar"},
			reject: []string{"", "foobar", "baz"},
		},
		{
			name:   "and",
			args:   []string{"foo", ",contains", "bar", ",contains", ",and"},
			match:  []string{"foobar", "barfoo", "a foo and a bar"},
			reject: []string{"", "foo", "bar"},
		},
		{
			name:   "star",
			args:   []string{"ab", ",star"},
			match:  []string{"", "ab", "abab"},
			reject: []string{"a", "aba", "b"},
		},
		{
			name:   "plus",
			args:   []string{"ab", ",plus"},
			match:  []string{"ab", "abab", "ababab"},
			reject: []string{"", "a", "aba", "b"},
		},
		{
			name:   "plus of a compound expression",
			args:   []string{"a", "b", ",or", ",plus"},
			match:  []string{"a", "b", "abba"},
			reject: []string{"", "abc"},
		},
		{
			name:   "not",
			args:   []string{"foo", ",not"},
			match:  []string{"", "bar", "foobar"},
			reject: []string{"foo"},
		},
		{
			name:   "contains",
			args:   []string{"foo", ",contains"},
			match:  []string{"foo", "foobar", "barfoo", "a foo b"},
			reject: []string{"", "fo", "bar"},
		},
		{
			name:   "contains the empty litteral",
			args:   []string{"", ",contains"},
			match:  []string{"", "anything"},
			reject: nil,
		},
		{
			// The generalized combination: what plain regular expressions cannot express.
			name:   "contains one litteral but not another",
			args:   []string{"foo", ",contains", "bar", ",contains", ",not", ",and"},
			match:  []string{"foo", "a foo b", "a foo and a baz"},
			reject: []string{"", "bar", "foobar", "a bar and then foo"},
		},
		{
			name:   "an operator applied to an operator's result",
			args:   []string{",dot", ",plus", " ", ",contains", ",not", ",and"},
			match:  []string{"abc", "é", "xé"},
			reject: []string{"", "a c", " "},
		},
		{
			name:   "operands are left on the stack in order",
			args:   []string{"a", "b", "c", ",concat", ",concat"},
			match:  []string{"abc"},
			reject: []string{"cba", "ab", "bc"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := filter.Parse(tc.args)
			if err != nil {
				t.Fatalf("Parse(%q) = _, %v, want no error", tc.args, err)
			}
			for _, s := range tc.match {
				if !gr.Match(r, []byte(s)) {
					t.Errorf("Parse(%q) = %v, which does not match %q", tc.args, r, s)
				}
			}
			for _, s := range tc.reject {
				if gr.Match(r, []byte(s)) {
					t.Errorf("Parse(%q) = %v, which matches %q", tc.args, r, s)
				}
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		// want is a substring of the expected error message.
		want string
	}{
		{
			name: "no argument at all",
			args: nil,
			want: "invalid length in stack",
		},
		{
			name: "two expressions left on the stack",
			args: []string{"a", "b"},
			want: "invalid length in stack",
		},
		{
			name: "a binary operator with an empty stack",
			args: []string{",concat"},
			want: `not enough arguments to "concat"`,
		},
		{
			name: "a binary operator with a single operand",
			args: []string{"a", ",or"},
			want: `not enough arguments to "or"`,
		},
		{
			name: "and with a single operand",
			args: []string{"a", ",and"},
			want: `not enough arguments to "and"`,
		},
		{
			name: "a unary operator with an empty stack",
			args: []string{",star"},
			want: `not enough arguments to "star"`,
		},
		{
			name: "plus with an empty stack",
			args: []string{",plus"},
			want: `not enough arguments to "plus"`,
		},
		{
			name: "not with an empty stack",
			args: []string{",not"},
			want: `not enough arguments to "not"`,
		},
		{
			name: "contains with an empty stack",
			args: []string{",contains"},
			want: `not enough arguments to "contains"`,
		},
		{
			name: "an unknown operator",
			args: []string{",bogus"},
			want: `unknown operator ",bogus"`,
		},
		{
			// A lone comma is an operator with an empty name, not a literal comma.
			name: "a lone comma",
			args: []string{","},
			want: `unknown operator ","`,
		},
		{
			// The operand is consumed before the failing operator is reported.
			name: "an unknown operator after a valid one",
			args: []string{"a", ",star", ",bogus"},
			want: `unknown operator ",bogus"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := filter.Parse(tc.args)
			if err == nil {
				t.Fatalf("Parse(%q) = %v, nil, want an error", tc.args, r)
			}
			if r != nil {
				t.Errorf("Parse(%q) = %v, want a nil expression alongside the error", tc.args, r)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Parse(%q) failed with %q, want it to mention %q", tc.args, err, tc.want)
			}
		})
	}
}

func TestLines(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		input string
		want  string
	}{
		{
			name:  "keeps the matching lines in order",
			args:  []string{"foo", ",contains"},
			input: "foo bar\nnothing\nbarfoo\n",
			want:  "foo bar\nbarfoo\n",
		},
		{
			name:  "keeps nothing",
			args:  []string{",empty"},
			input: "a\nb\n",
			want:  "",
		},
		{
			name:  "keeps everything",
			args:  []string{",any"},
			input: "a\n\nb\n",
			want:  "a\n\nb\n",
		},
		{
			name:  "matches whole lines only",
			args:  []string{"foo"},
			input: "foo\nfoobar\na foo b\n",
			want:  "foo\n",
		},
		{
			name:  "matches the empty line",
			args:  []string{""},
			input: "\na\n\n",
			want:  "\n\n",
		},
		{
			name:  "reads an input with no trailing newline",
			args:  []string{"foo", ",contains"},
			input: "bar\nfoo",
			want:  "foo\n",
		},
		{
			name:  "reads an empty input",
			args:  []string{",any"},
			input: "",
			want:  "",
		},
		{
			name:  "strips the carriage return of CRLF input",
			args:  []string{"foo"},
			input: "foo\r\nbar\r\n",
			want:  "foo\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := filter.Parse(tc.args)
			if err != nil {
				t.Fatalf("Parse(%q) = _, %v, want no error", tc.args, err)
			}
			var out bytes.Buffer
			if err := filter.Lines(r, strings.NewReader(tc.input), &out); err != nil {
				t.Fatalf("Lines(%v, %q, _) = %v, want no error", r, tc.input, err)
			}
			if got := out.String(); got != tc.want {
				t.Errorf("Lines(%v, %q, _) wrote %q, want %q", r, tc.input, got, tc.want)
			}
		})
	}
}

// errWriter fails every write, like a closed pipe would.
type errWriter struct{ err error }

func (w errWriter) Write([]byte) (int, error) { return 0, w.err }

func TestLinesReportsAWriteError(t *testing.T) {
	r, err := filter.Parse([]string{",any"})
	if err != nil {
		t.Fatalf("Parse() = _, %v, want no error", err)
	}

	want := errors.New("no room left")
	if got := filter.Lines(r, strings.NewReader("a\nb\n"), errWriter{err: want}); !errors.Is(got, want) {
		t.Errorf("Lines() = %v, want %v", got, want)
	}
}

func TestLinesReportsAReadError(t *testing.T) {
	r, err := filter.Parse([]string{",any"})
	if err != nil {
		t.Fatalf("Parse() = _, %v, want no error", err)
	}

	want := errors.New("disk on fire")
	var out bytes.Buffer
	got := filter.Lines(r, iotest.ErrReader(want), &out)
	if !errors.Is(got, want) {
		t.Errorf("Lines() = %v, want it to wrap %v", got, want)
	}
}

// A line too long for the scanner must be reported rather than silently ending the scan: doing so
// used to drop that line and every line after it while still reporting success.
func TestLinesReportsAnOverlongLine(t *testing.T) {
	r, err := filter.Parse([]string{"foo", ",contains"})
	if err != nil {
		t.Fatalf("Parse() = _, %v, want no error", err)
	}

	input := strings.Repeat("a", bufio.MaxScanTokenSize+1) + "\nfoo\n"
	var out bytes.Buffer
	got := filter.Lines(r, strings.NewReader(input), &out)
	if !errors.Is(got, bufio.ErrTooLong) {
		t.Errorf("Lines() = %v, want it to wrap %v", got, bufio.ErrTooLong)
	}
	// The lines that follow are never reached, so none of them may be reported as matching.
	if out.Len() != 0 {
		t.Errorf("Lines() wrote %q, want nothing once the scan has failed", out.String())
	}
}

// A reader that never matches must not be written to at all, so a failing writer goes unnoticed.
func TestLinesDoesNotWriteWhenNothingMatches(t *testing.T) {
	r, err := filter.Parse([]string{",empty"})
	if err != nil {
		t.Fatalf("Parse() = _, %v, want no error", err)
	}

	if got := filter.Lines(r, strings.NewReader("a\nb\n"), errWriter{err: errors.New("no room left")}); got != nil {
		t.Errorf("Lines() = %v, want no error", got)
	}
}
