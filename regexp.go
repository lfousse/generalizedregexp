// Package generalizedregexp implement regular expression extended with `and` and `not` using the Brzozowski derivative.
package generalizedregexp

import (
	"fmt"
	"unicode/utf8"
)

// Regexp implements generalized regexps.
type Regexp interface {
	// Derive returns a new regexp such that: For a rune c and word w, r.Derive(c) matches w iff r matches c.w.
	Derive(rune) Regexp
	// Nu returns true iff the regexp matches the empty word.
	Nu() bool
	// String returns an s-exp encoding of the regexp, mostly for debugging.
	String() string
	// Provably empty returns true when we know the Regexp matches no word. False negatives are possible.
	provablyEmpty() bool
}

// maybeNotEmpty is used as the default implementation for `provablyEmpty`.
type maybeNotEmpty struct{}

func (maybeNotEmpty) provablyEmpty() bool { return false }

// emptySetImpl matches no word.
type emptySetImpl struct{}

// EmptySet returns a Regexp that matches no word.
func EmptySet() Regexp {
	return emptySetImpl{}
}

func (emptySetImpl) Derive(_ rune) Regexp {
	return emptySetImpl{}
}

func (emptySetImpl) Nu() bool {
	return false
}

func (emptySetImpl) String() string {
	return "ø"
}

func (emptySetImpl) provablyEmpty() bool { return true }

type epsilonImpl struct {
	maybeNotEmpty
}

// Epsilon returns a regexp that matches only the empty string.
func Epsilon() Regexp { return epsilonImpl{} }

func (epsilonImpl) Derive(_ rune) Regexp {
	return emptySetImpl{}
}

func (epsilonImpl) Nu() bool {
	return true
}

func (epsilonImpl) String() string {
	return "ε"
}

type litteralStringImpl struct {
	maybeNotEmpty

	s []byte
}

func (l litteralStringImpl) String() string {
	return fmt.Sprintf("%q", string(l.s))
}

// LitteralString returns a regexp that matches exactly one string.
func LitteralString(s string) Regexp {
	return &litteralStringImpl{
		s: []byte(s),
	}
}

func (l litteralStringImpl) Derive(r rune) Regexp {
	if len(l.s) == 0 {
		// utf8.DecodeRune would report (RuneError, 0) here, which must not be mistaken for an
		// actual U+FFFD rune.
		return emptySetImpl{}
	}
	a, size := utf8.DecodeRune(l.s)
	if a != r {
		return emptySetImpl{}
	}
	if size == len(l.s) {
		return epsilonImpl{}
	}
	return &litteralStringImpl{
		s: l.s[size:],
	}
}

func (l litteralStringImpl) Nu() bool {
	return len(l.s) == 0
}

type orImpl struct {
	maybeNotEmpty

	a, b Regexp
}

// matchesAll checks whether `r` matches all words (".*"). It is used internally to simplify disjunctions after derivation.
func matchesAll(r Regexp) bool {
	s, ok := r.(*starImpl)
	if !ok {
		return false
	}
	_, ok = s.c.(anyRuneImpl)
	return ok
}

// Or returns a regexp matching the union of the languages matched by each input regexp.
func Or(a, b Regexp) Regexp {
	if matchesAll(a) {
		return a
	}
	if matchesAll(b) {
		return b
	}

	return &orImpl{
		a: a,
		b: b,
	}
}

func (o orImpl) Derive(r rune) Regexp {
	a := o.a.Derive(r)
	if a.provablyEmpty() {
		return o.b.Derive(r)
	}
	b := o.b.Derive(r)
	if b.provablyEmpty() {
		return a
	}
	return Or(a, b)
}

func (o orImpl) Nu() bool {
	return o.a.Nu() || o.b.Nu()
}

func (o orImpl) String() string {
	return fmt.Sprintf("(or %v %v)", o.a, o.b)
}

type concatImpl struct {
	maybeNotEmpty

	r, s Regexp
}

// Concat is self-explanatory.
func Concat(r, s Regexp) Regexp {
	if _, ok := r.(epsilonImpl); ok {
		return s
	}
	if _, ok := s.(epsilonImpl); ok {
		return r
	}
	return &concatImpl{
		r: r,
		s: s,
	}
}

func (c concatImpl) Derive(r rune) Regexp {
	dr := c.r.Derive(r)
	if dr.provablyEmpty() {
		if c.r.Nu() {
			return c.s.Derive(r)
		}
		return EmptySet()
	}
	if !c.r.Nu() {
		return Concat(dr, c.s)
	}
	ds := c.s.Derive(r)
	if ds.provablyEmpty() {
		return Concat(dr, c.s)
	}
	return Or(ds, Concat(dr, c.s))
}

func (c concatImpl) Nu() bool {
	return c.r.Nu() && c.s.Nu()
}

func (c concatImpl) String() string {
	return fmt.Sprintf("(concat %v %v)", c.r, c.s)
}

type starImpl struct {
	maybeNotEmpty

	c Regexp
}

// Star implements Kleene star operation.
func Star(c Regexp) Regexp {
	return &starImpl{c: c}
}

func (s starImpl) Derive(r rune) Regexp {
	return Concat(s.c.Derive(r), s)
}

func (s starImpl) Nu() bool {
	return true
}

func (s starImpl) String() string {
	return fmt.Sprintf("(star %v)", s.c)
}

type andImpl struct {
	maybeNotEmpty

	a, b Regexp
}

// And returns a regexp matching the intersection of the languages matched by each input regexp.
func And(a, b Regexp) Regexp {
	if matchesAll(a) {
		return b
	}
	if matchesAll(b) {
		return a
	}
	return &andImpl{
		a: a,
		b: b,
	}
}

func (a andImpl) Derive(r rune) Regexp {
	return And(a.a.Derive(r), a.b.Derive(r))
}

func (a andImpl) Nu() bool {
	return a.a.Nu() && a.b.Nu()
}

func (a andImpl) String() string {
	return fmt.Sprintf("(and %v %v)", a.a, a.b)
}

type notImpl struct {
	maybeNotEmpty

	a Regexp
}

// Not returns a regexp matching the complement of the language matched by the input.
func Not(r Regexp) Regexp {
	return &notImpl{a: r}
}

func (n notImpl) Derive(r rune) Regexp {
	return Not(n.a.Derive(r))
}

func (n notImpl) Nu() bool {
	return !n.a.Nu()
}

func (n notImpl) String() string {
	return fmt.Sprintf("(not %v)", n.a)
}

type anyRuneImpl struct {
	maybeNotEmpty
}

func (anyRuneImpl) Derive(_ rune) Regexp {
	return epsilonImpl{}
}

func (anyRuneImpl) Nu() bool {
	return false
}

func (anyRuneImpl) String() string {
	return "."
}

// AnyRune returns a regexp that matches any single rune.
func AnyRune() Regexp { return anyRuneImpl{} }

// AnyString returns a regexp that matches any string (typically represented as ".*").
func AnyString() Regexp {
	return Star(AnyRune())
}

// Contains is a convenience wrapper for ".*s.*".
func Contains(s string) Regexp {
	return Concat(AnyString(), Concat(LitteralString(s), AnyString()))
}

// Match returns true iff the string s is matched by the regexp r.
func Match(r Regexp, s []byte) bool {
	for {
		if len(s) == 0 {
			return r.Nu()
		}
		a, size := utf8.DecodeRune(s)
		r = r.Derive(a)
		s = s[size:]
	}
}
