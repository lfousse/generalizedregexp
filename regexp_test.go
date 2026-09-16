package generalizedregexp_test

import (
	"math/rand"
	"regexp"
	"strings"
	"testing"

	gr "github.com/lfousse/generalizedregexp"
)

// checkMatch asserts that r matches (or not) s, and that the complement of r behaves the opposite way.
func checkMatch(t *testing.T, r gr.Regexp, s string, want bool) {
	t.Helper()
	if got := gr.Match(r, []byte(s)); got != want {
		t.Errorf("Match(%v, %q) = %v, want %v", r, s, got, want)
	}
	if got := gr.Match(gr.Not(r), []byte(s)); got == want {
		t.Errorf("Match(%v, %q) = %v, want %v", gr.Not(r), s, got, !want)
	}
}

func TestLitteral(t *testing.T) {
	tests := []struct {
		r    string
		s    string
		want bool
	}{
		{
			r:    "",
			s:    "",
			want: true,
		},
		{
			r: "",
			s: "a",
		},
		{
			r: "a",
			s: "",
		},
		{
			r:    "foo",
			s:    "foo",
			want: true,
		},
		{
			r: "foo",
			s: "bla foo",
		},
	}

	for _, tc := range tests {
		r := gr.LitteralString(tc.r)
		if got := gr.Match(r, []byte(tc.s)); got != tc.want {
			t.Errorf("Match(%q, %q) = %v, want = %v", tc.r, tc.s, got, tc.want)
		}

		// Negation test for free.
		want := !tc.want
		nr := gr.Not(gr.LitteralString(tc.r))
		if got := gr.Match(nr, []byte(tc.s)); got != want {
			t.Errorf("Match(%q, %q) = %v, want = %v", nr, tc.s, got, want)
		}
	}
}

func TestEmptySet(t *testing.T) {
	tests := []struct {
		s string
	}{
		{
			s: "",
		},
		{
			s: "a",
		},
		{
			s: "foo",
		},
		{
			s: "bla foo",
		},
	}

	for _, tc := range tests {
		r := gr.EmptySet()
		want := false
		if got := gr.Match(r, []byte(tc.s)); got {
			t.Errorf("Match(%v, %q) = %v, want = %v", r, tc.s, got, want)
		}

		// Negation test for free.
		nr := gr.Not(gr.EmptySet())
		if got := gr.Match(nr, []byte(tc.s)); !got {
			t.Errorf("Match(%v, %q) = %v, want = %v", nr, tc.s, got, true)
		}
	}
}

func TestEpsilon(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{
			s:    "",
			want: true,
		},
		{
			s: "a",
		},
		{
			s: "foo",
		},
		{
			s: "bla foo",
		},
	}

	for _, tc := range tests {
		r := gr.Epsilon()
		if got := gr.Match(r, []byte(tc.s)); got != tc.want {
			t.Errorf("Match(%v, %q) = %v, want = %v", r, tc.s, got, tc.want)
		}

		// Negation test for free.
		nr := gr.Not(gr.Epsilon())
		if got := gr.Match(nr, []byte(tc.s)); got == tc.want {
			t.Errorf("Match(%v, %q) = %v, want = %v", nr, tc.s, got, !tc.want)
		}
	}
}

func TestAnd(t *testing.T) {
	tests := []struct {
		inc  string
		exc  string
		s    string
		want bool
	}{
		{
			inc: "foo",
			exc: "bar",
			s:   "",
		},
		{
			inc:  "foo",
			exc:  "bar",
			s:    "foo",
			want: true,
		},
		{
			inc:  "foo",
			exc:  "bar",
			s:    "something foo",
			want: true,
		},
		{
			inc:  "foo",
			exc:  "bar",
			s:    "foosomething",
			want: true,
		},
		{
			inc: "foo",
			exc: "bar",
			s:   "bar",
		},
		{
			inc: "foo",
			exc: "bar",
			s:   "foobar",
		},
		{
			inc: "foo",
			exc: "bar",
			s:   "something bar and then foo",
		},
	}

	for _, tc := range tests {
		r := gr.And(gr.Contains(tc.inc), gr.Not(gr.Contains(tc.exc)))
		if got := gr.Match(r, []byte(tc.s)); got != tc.want {
			t.Errorf("Match(inc=%q, exc=%q, %q) = %v, want %v", tc.inc, tc.exc, tc.s, got, tc.want)
		}
	}
}

func TestOr(t *testing.T) {
	tests := []struct {
		name string
		r    gr.Regexp
		s    string
		want bool
	}{
		{
			name: "left branch",
			r:    gr.Or(gr.LitteralString("foo"), gr.LitteralString("bar")),
			s:    "foo",
			want: true,
		},
		{
			name: "right branch",
			r:    gr.Or(gr.LitteralString("foo"), gr.LitteralString("bar")),
			s:    "bar",
			want: true,
		},
		{
			// Both branches survive the first derivation: "ba" is a prefix of both.
			name: "common prefix",
			r:    gr.Or(gr.LitteralString("bar"), gr.LitteralString("baz")),
			s:    "baz",
			want: true,
		},
		{
			name: "no branch",
			r:    gr.Or(gr.LitteralString("foo"), gr.LitteralString("bar")),
			s:    "foobar",
		},
		{
			name: "empty word, no branch nullable",
			r:    gr.Or(gr.LitteralString("foo"), gr.LitteralString("bar")),
			s:    "",
		},
		{
			name: "empty word, one branch nullable",
			r:    gr.Or(gr.Epsilon(), gr.LitteralString("bar")),
			s:    "",
			want: true,
		},
		{
			name: "with the empty set",
			r:    gr.Or(gr.EmptySet(), gr.LitteralString("foo")),
			s:    "foo",
			want: true,
		},
		{
			// Or absorbs everything into ".*" (see matchesAll); the result must still match anything.
			name: "anystring on the left absorbs",
			r:    gr.Or(gr.AnyString(), gr.LitteralString("foo")),
			s:    "anything at all",
			want: true,
		},
		{
			name: "anystring on the right absorbs",
			r:    gr.Or(gr.LitteralString("foo"), gr.AnyString()),
			s:    "",
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			checkMatch(t, tc.r, tc.s, tc.want)
		})
	}
}

func TestConcat(t *testing.T) {
	tests := []struct {
		name string
		r    gr.Regexp
		s    string
		want bool
	}{
		{
			name: "two litterals",
			r:    gr.Concat(gr.LitteralString("foo"), gr.LitteralString("bar")),
			s:    "foobar",
			want: true,
		},
		{
			name: "two litterals, wrong order",
			r:    gr.Concat(gr.LitteralString("foo"), gr.LitteralString("bar")),
			s:    "barfoo",
		},
		{
			name: "epsilon is a left unit",
			r:    gr.Concat(gr.Epsilon(), gr.LitteralString("foo")),
			s:    "foo",
			want: true,
		},
		{
			name: "epsilon is a right unit",
			r:    gr.Concat(gr.LitteralString("foo"), gr.Epsilon()),
			s:    "foo",
			want: true,
		},
		{
			name: "the empty set annihilates",
			r:    gr.Concat(gr.LitteralString("foo"), gr.EmptySet()),
			s:    "foo",
		},
		{
			// The left factor is nullable, so the derivative must consider both factors.
			name: "nullable left factor, skipped",
			r:    gr.Concat(gr.Star(gr.LitteralString("a")), gr.LitteralString("b")),
			s:    "b",
			want: true,
		},
		{
			name: "nullable left factor, used",
			r:    gr.Concat(gr.Star(gr.LitteralString("a")), gr.LitteralString("b")),
			s:    "aaab",
			want: true,
		},
		{
			// Both factors can consume the same rune: the split point is ambiguous.
			name: "ambiguous split",
			r:    gr.Concat(gr.Star(gr.LitteralString("a")), gr.Star(gr.LitteralString("a"))),
			s:    "aaaa",
			want: true,
		},
		{
			name: "both factors nullable, empty word",
			r:    gr.Concat(gr.Star(gr.LitteralString("a")), gr.Star(gr.LitteralString("b"))),
			s:    "",
			want: true,
		},
		{
			name: "both factors nullable, wrong order",
			r:    gr.Concat(gr.Star(gr.LitteralString("a")), gr.Star(gr.LitteralString("b"))),
			s:    "ba",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			checkMatch(t, tc.r, tc.s, tc.want)
		})
	}
}

func TestStar(t *testing.T) {
	tests := []struct {
		name string
		r    gr.Regexp
		s    string
		want bool
	}{
		{
			name: "empty word",
			r:    gr.Star(gr.LitteralString("ab")),
			s:    "",
			want: true,
		},
		{
			name: "one repetition",
			r:    gr.Star(gr.LitteralString("ab")),
			s:    "ab",
			want: true,
		},
		{
			name: "several repetitions",
			r:    gr.Star(gr.LitteralString("ab")),
			s:    "ababab",
			want: true,
		},
		{
			name: "truncated repetition",
			r:    gr.Star(gr.LitteralString("ab")),
			s:    "ababa",
		},
		{
			// Requires backtracking-free handling of an ambiguous factorization: "aab" is
			// "a"+"ab" but not "a"+"a"+"b".
			name: "ambiguous alternation under star",
			r:    gr.Star(gr.Or(gr.LitteralString("a"), gr.LitteralString("ab"))),
			s:    "aab",
			want: true,
		},
		{
			name: "star of epsilon matches only the empty word",
			r:    gr.Star(gr.Epsilon()),
			s:    "",
			want: true,
		},
		{
			name: "star of epsilon rejects",
			r:    gr.Star(gr.Epsilon()),
			s:    "a",
		},
		{
			name: "star of the empty set matches the empty word",
			r:    gr.Star(gr.EmptySet()),
			s:    "",
			want: true,
		},
		{
			name: "star of the empty set rejects",
			r:    gr.Star(gr.EmptySet()),
			s:    "a",
		},
		{
			name: "nested stars",
			r:    gr.Star(gr.Star(gr.LitteralString("a"))),
			s:    "aaa",
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			checkMatch(t, tc.r, tc.s, tc.want)
		})
	}
}

func TestAnyRune(t *testing.T) {
	tests := []struct {
		name string
		r    gr.Regexp
		s    string
		want bool
	}{
		{
			name: "one ascii rune",
			r:    gr.AnyRune(),
			s:    "a",
			want: true,
		},
		{
			// A multi-byte rune is a single rune: "." must consume all of it.
			name: "one multi-byte rune",
			r:    gr.AnyRune(),
			s:    "é",
			want: true,
		},
		{
			name: "one astral rune",
			r:    gr.AnyRune(),
			s:    "🙂",
			want: true,
		},
		{
			name: "the empty word is not a rune",
			r:    gr.AnyRune(),
			s:    "",
		},
		{
			name: "two runes",
			r:    gr.AnyRune(),
			s:    "ab",
		},
		{
			name: "exactly two runes",
			r:    gr.Concat(gr.AnyRune(), gr.AnyRune()),
			s:    "éé",
			want: true,
		},
		{
			// "é" is two bytes but one rune, so it must not match ".." .
			name: "two bytes are not two runes",
			r:    gr.Concat(gr.AnyRune(), gr.AnyRune()),
			s:    "é",
		},
		{
			name: "anystring matches the empty word",
			r:    gr.AnyString(),
			s:    "",
			want: true,
		},
		{
			name: "anystring matches anything",
			r:    gr.AnyString(),
			s:    "héllo, wörld 🙂",
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			checkMatch(t, tc.r, tc.s, tc.want)
		})
	}
}

func TestUnicode(t *testing.T) {
	tests := []struct {
		name string
		r    gr.Regexp
		s    string
		want bool
	}{
		{
			name: "multi-byte litteral",
			r:    gr.LitteralString("héllo"),
			s:    "héllo",
			want: true,
		},
		{
			name: "multi-byte litteral, one rune off",
			r:    gr.LitteralString("héllo"),
			s:    "hello",
		},
		{
			// Same first byte (0xC3) but a different rune: deriving must compare runes, not bytes.
			name: "runes sharing a leading byte",
			r:    gr.LitteralString("é"),
			s:    "è",
		},
		{
			name: "astral litteral",
			r:    gr.LitteralString("🙂🙁"),
			s:    "🙂🙁",
			want: true,
		},
		{
			name: "astral litteral, truncated",
			r:    gr.LitteralString("🙂🙁"),
			s:    "🙂",
		},
		{
			name: "contains a multi-byte litteral",
			r:    gr.Contains("é"),
			s:    "café au lait",
			want: true,
		},
		{
			// Decomposed form: "e" + U+0301 is not the precomposed rune U+00E9.
			name: "contains, decomposed form does not match",
			r:    gr.Contains("é"),
			s:    "café au lait",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			checkMatch(t, tc.r, tc.s, tc.want)
		})
	}
}

func TestEmptyLitteralString(t *testing.T) {
	r := gr.LitteralString("")

	// The empty litteral is equivalent to epsilon: it matches the empty word and nothing else.
	checkMatch(t, r, "", true)
	checkMatch(t, r, "a", false)
	// utf8.DecodeRune on an empty slice reports (RuneError, 0), which must not be mistaken
	// for an actual U+FFFD rune.
	checkMatch(t, r, "�", false)
}

func TestContains(t *testing.T) {
	tests := []struct {
		name string
		sub  string
		s    string
		want bool
	}{
		{
			name: "prefix",
			sub:  "foo",
			s:    "foobar",
			want: true,
		},
		{
			name: "suffix",
			sub:  "bar",
			s:    "foobar",
			want: true,
		},
		{
			name: "infix",
			sub:  "oob",
			s:    "foobar",
			want: true,
		},
		{
			name: "whole word",
			sub:  "foobar",
			s:    "foobar",
			want: true,
		},
		{
			name: "absent",
			sub:  "baz",
			s:    "foobar",
		},
		{
			// A partial occurrence must not be enough, and the search has to restart after it.
			name: "restart after a partial occurrence",
			sub:  "aab",
			s:    "aaab",
			want: true,
		},
		{
			name: "overlapping false start",
			sub:  "aab",
			s:    "aaa",
		},
		{
			name: "the empty substring is everywhere",
			sub:  "",
			s:    "foobar",
			want: true,
		},
		{
			name: "the empty substring is in the empty word",
			sub:  "",
			s:    "",
			want: true,
		},
		{
			name: "nothing is contained in the empty word",
			sub:  "a",
			s:    "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			checkMatch(t, gr.Contains(tc.sub), tc.s, tc.want)
		})
	}
}

// sampleRegexps returns regexps exercising every constructor, for the algebraic laws below.
func sampleRegexps() map[string]gr.Regexp {
	return map[string]gr.Regexp{
		"emptyset":  gr.EmptySet(),
		"epsilon":   gr.Epsilon(),
		"litteral":  gr.LitteralString("ab"),
		"anyrune":   gr.AnyRune(),
		"anystring": gr.AnyString(),
		"or":        gr.Or(gr.LitteralString("a"), gr.LitteralString("ba")),
		"concat":    gr.Concat(gr.Star(gr.LitteralString("a")), gr.LitteralString("b")),
		"star":      gr.Star(gr.Or(gr.LitteralString("a"), gr.LitteralString("ab"))),
		"and":       gr.And(gr.Contains("a"), gr.Not(gr.Contains("bb"))),
		"not":       gr.Not(gr.LitteralString("ab")),
		"contains":  gr.Contains("ab"),
	}
}

// sampleWords returns short words over {a, b} plus a few odd ones.
func sampleWords() []string {
	words := []string{"", "c", "é"}
	for i := 0; i < 32; i++ {
		var w strings.Builder
		for j := i; j > 0; j /= 2 {
			w.WriteByte("ab"[j%2])
		}
		words = append(words, w.String())
	}
	return words
}

// TestNuAgreesWithMatch checks the documented contract of Nu: it holds iff the empty word matches.
func TestNuAgreesWithMatch(t *testing.T) {
	for name, r := range sampleRegexps() {
		t.Run(name, func(t *testing.T) {
			if got, want := r.Nu(), gr.Match(r, nil); got != want {
				t.Errorf("(%v).Nu() = %v, but Match(%v, \"\") = %v", r, got, r, want)
			}
		})
	}
}

// TestDeriveIsResidual checks the defining property of the Brzozowski derivative:
// r.Derive(c) matches w iff r matches c.w.
func TestDeriveIsResidual(t *testing.T) {
	for name, r := range sampleRegexps() {
		t.Run(name, func(t *testing.T) {
			for _, c := range []rune{'a', 'b', 'c', 'é'} {
				d := r.Derive(c)
				for _, w := range sampleWords() {
					got := gr.Match(d, []byte(w))
					want := gr.Match(r, []byte(string(c)+w))
					if got != want {
						t.Errorf("Match(%v.Derive(%q), %q) = %v, want %v (as Match(%v, %q) = %v)",
							r, c, w, got, want, r, string(c)+w, want)
					}
				}
			}
		})
	}
}

// TestDeMorgan checks that And, Or and Not agree on the boolean laws they are supposed to satisfy.
func TestDeMorgan(t *testing.T) {
	samples := sampleRegexps()
	words := sampleWords()

	for aName, a := range samples {
		for bName, b := range samples {
			t.Run(aName+"/"+bName, func(t *testing.T) {
				for _, w := range words {
					s := []byte(w)

					if got, want := gr.Match(gr.Not(gr.Or(a, b)), s), gr.Match(gr.And(gr.Not(a), gr.Not(b)), s); got != want {
						t.Errorf("¬(%v ∪ %v) matches %q = %v, but ¬%v ∩ ¬%v = %v", a, b, w, got, a, b, want)
					}
					if got, want := gr.Match(gr.Not(gr.And(a, b)), s), gr.Match(gr.Or(gr.Not(a), gr.Not(b)), s); got != want {
						t.Errorf("¬(%v ∩ %v) matches %q = %v, but ¬%v ∪ ¬%v = %v", a, b, w, got, a, b, want)
					}
					// Union and intersection are commutative.
					if got, want := gr.Match(gr.Or(a, b), s), gr.Match(gr.Or(b, a), s); got != want {
						t.Errorf("(%v ∪ %v) matches %q = %v, but (%v ∪ %v) = %v", a, b, w, got, b, a, want)
					}
					if got, want := gr.Match(gr.And(a, b), s), gr.Match(gr.And(b, a), s); got != want {
						t.Errorf("(%v ∩ %v) matches %q = %v, but (%v ∩ %v) = %v", a, b, w, got, b, a, want)
					}
				}
			})
		}
	}
}

// TestDoubleNegation checks that Not is an involution, and that the neutral and absorbing
// elements of the algebra are what they claim to be.
func TestDoubleNegation(t *testing.T) {
	for name, r := range sampleRegexps() {
		t.Run(name, func(t *testing.T) {
			for _, w := range sampleWords() {
				s := []byte(w)

				if got, want := gr.Match(gr.Not(gr.Not(r)), s), gr.Match(r, s); got != want {
					t.Errorf("¬¬%v matches %q = %v, want %v", r, w, got, want)
				}
				if got := gr.Match(gr.Or(r, gr.Not(r)), s); !got {
					t.Errorf("%v ∪ ¬%v matches %q = false, want true", r, r, w)
				}
				if got := gr.Match(gr.And(r, gr.Not(r)), s); got {
					t.Errorf("%v ∩ ¬%v matches %q = true, want false", r, r, w)
				}
				// ".*" is neutral for ∩ and absorbing for ∪; ø is the converse.
				if got, want := gr.Match(gr.And(gr.AnyString(), r), s), gr.Match(r, s); got != want {
					t.Errorf(".* ∩ %v matches %q = %v, want %v", r, w, got, want)
				}
				if got, want := gr.Match(gr.Or(gr.EmptySet(), r), s), gr.Match(r, s); got != want {
					t.Errorf("ø ∪ %v matches %q = %v, want %v", r, w, got, want)
				}
				if got := gr.Match(gr.Or(gr.AnyString(), r), s); !got {
					t.Errorf(".* ∪ %v matches %q = false, want true", r, w)
				}
				if got := gr.Match(gr.And(gr.EmptySet(), r), s); got {
					t.Errorf("ø ∩ %v matches %q = true, want false", r, w)
				}
			}
		})
	}
}

// TestAgainstStdlib cross-checks the ∩/¬-free fragment against the standard library, on random
// regexps and random words.
func TestAgainstStdlib(t *testing.T) {
	rng := rand.New(rand.NewSource(1))

	// build returns a random regexp together with an equivalent stdlib pattern.
	var build func(depth int) (gr.Regexp, string)
	build = func(depth int) (gr.Regexp, string) {
		if depth == 0 {
			switch rng.Intn(4) {
			case 0:
				return gr.Epsilon(), ""
			case 1:
				return gr.AnyRune(), "."
			case 2:
				return gr.LitteralString("a"), "a"
			default:
				return gr.LitteralString("b"), "b"
			}
		}
		l, lp := build(depth - 1)
		r, rp := build(depth - 1)
		switch rng.Intn(3) {
		case 0:
			return gr.Or(l, r), "(?:" + lp + "|" + rp + ")"
		case 1:
			return gr.Concat(l, r), "(?:" + lp + ")(?:" + rp + ")"
		default:
			return gr.Star(l), "(?:" + lp + ")*"
		}
	}

	words := sampleWords()
	for i := 0; i < 200; i++ {
		r, pattern := build(1 + rng.Intn(3))
		want := regexp.MustCompile("^(?:" + pattern + ")$")
		for _, w := range words {
			if got, exp := gr.Match(r, []byte(w)), want.MatchString(w); got != exp {
				t.Errorf("Match(%v, %q) = %v, want %v (stdlib pattern %q)", r, w, got, exp, pattern)
			}
		}
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		r    gr.Regexp
		want string
	}{
		{r: gr.EmptySet(), want: "ø"},
		{r: gr.Epsilon(), want: "ε"},
		{r: gr.AnyRune(), want: "."},
		{r: gr.AnyString(), want: "(star .)"},
		{r: gr.LitteralString("fo\"o"), want: `"fo\"o"`},
		{r: gr.Or(gr.LitteralString("a"), gr.Epsilon()), want: `(or "a" ε)`},
		{r: gr.And(gr.LitteralString("a"), gr.Epsilon()), want: `(and "a" ε)`},
		{r: gr.Not(gr.LitteralString("a")), want: `(not "a")`},
		{r: gr.Star(gr.LitteralString("a")), want: `(star "a")`},
		{r: gr.Concat(gr.LitteralString("a"), gr.LitteralString("b")), want: `(concat "a" "b")`},
		// Concat drops epsilon, Or and And collapse ".*".
		{r: gr.Concat(gr.Epsilon(), gr.LitteralString("a")), want: `"a"`},
		{r: gr.Concat(gr.LitteralString("a"), gr.Epsilon()), want: `"a"`},
		{r: gr.Or(gr.AnyString(), gr.LitteralString("a")), want: "(star .)"},
		{r: gr.And(gr.AnyString(), gr.LitteralString("a")), want: `"a"`},
		{r: gr.Contains("a"), want: `(concat (star .) (concat "a" (star .)))`},
	}

	for _, tc := range tests {
		if got := tc.r.String(); got != tc.want {
			t.Errorf("String() = %v, want %v", got, tc.want)
		}
	}
}

// TestLongInput guards against the derivative growing without bound on a long input.
func TestLongInput(t *testing.T) {
	r := gr.And(gr.Contains("foo"), gr.Not(gr.Contains("bar")))
	s := strings.Repeat("foa bao ", 2000) + "foo"

	if got := gr.Match(r, []byte(s)); !got {
		t.Errorf("Match(%v, <long input>) = false, want true", r)
	}
	if got := gr.Match(r, []byte(s+"bar")); got {
		t.Errorf("Match(%v, <long input with bar>) = true, want false", r)
	}
}
