// Package filter parses generalized regular expressions written in postfix notation, and selects
// the lines of a text that they match.
package filter

import (
	"bufio"
	"fmt"
	"io"

	gr "github.com/lfousse/generalizedregexp"
)

// Parse builds a regexp out of a generalized regular expression written in postfix notation.
//
// Each argument is either an operator, written as a comma followed by its name, or a literal
// string. Literals are pushed onto a stack, and operators pop their operands off it; exactly one
// expression must be left on the stack once every argument has been read. To write a literal
// starting with a comma, double it: ",,star" is the literal ",star", whereas ",star" is the
// repetition operator.
//
// The operators are ",concat", ",or", ",and", ",not", ",star", ",plus" and ",contains", along with
// the constants ",dot" (any single rune), ",any" (any string) and ",empty" (the empty set).
func Parse(rpn []string) (gr.Regexp, error) {
	var stack []gr.Regexp

	for _, a := range rpn {
		if len(a) == 0 {
			stack = append(stack, gr.LitteralString(a))
			continue
		}
		if len(a) >= 2 && a[0] == ',' && a[1] == ',' {
			stack = append(stack, gr.LitteralString(a[1:]))
			continue
		}
		if a[0] != ',' {
			stack = append(stack, gr.LitteralString(a))
			continue
		}
		a = a[1:]
		switch a {
		case "any":
			stack = append(stack, gr.AnyString())

		case "empty":
			stack = append(stack, gr.EmptySet())

		case "dot":
			stack = append(stack, gr.AnyRune())

		case "concat":
			if len(stack) < 2 {
				return nil, fmt.Errorf("not enough arguments to %q", a)
			}
			n := len(stack)
			c := gr.Concat(stack[n-2], stack[n-1])
			stack = append(stack[:n-2], c)

		case "or":
			if len(stack) < 2 {
				return nil, fmt.Errorf("not enough arguments to %q", a)
			}
			n := len(stack)
			c := gr.Or(stack[n-2], stack[n-1])
			stack = append(stack[:n-2], c)

		case "and":
			if len(stack) < 2 {
				return nil, fmt.Errorf("not enough arguments to %q", a)
			}
			n := len(stack)
			c := gr.And(stack[n-2], stack[n-1])
			stack = append(stack[:n-2], c)

		case "star":
			if len(stack) < 1 {
				return nil, fmt.Errorf("not enough arguments to %q", a)
			}
			n := len(stack)
			c := gr.Star(stack[n-1])
			stack = append(stack[:n-1], c)

		case "plus":
			if len(stack) < 1 {
				return nil, fmt.Errorf("not enough arguments to %q", a)
			}
			n := len(stack)
			c := gr.Concat(stack[n-1], gr.Star(stack[n-1]))
			stack = append(stack[:n-1], c)

		case "not":
			if len(stack) < 1 {
				return nil, fmt.Errorf("not enough arguments to %q", a)
			}
			n := len(stack)
			c := gr.Not(stack[n-1])
			stack = append(stack[:n-1], c)

		case "contains":
			if len(stack) < 1 {
				return nil, fmt.Errorf("not enough arguments to %q", a)
			}
			n := len(stack)
			c := gr.Concat(gr.AnyString(), gr.Concat(stack[n-1], gr.AnyString()))
			stack = append(stack[:n-1], c)

		default:
			return nil, fmt.Errorf("unknown operator %q (use \",,\" to escape a leading comma)", ","+a)
		}
	}

	if len(stack) != 1 {
		return nil, fmt.Errorf("invalid length in stack: %v", stack)
	}
	return stack[0], nil
}

// Lines writes to out every line read from in that r matches in its entirety. It reports an error
// if in cannot be read to the end, which notably happens on a line longer than the maximum token
// size of a bufio.Scanner, or if out cannot be written to.
func Lines(r gr.Regexp, in io.Reader, out io.Writer) error {
	lines := bufio.NewScanner(in)
	for lines.Scan() {
		l := lines.Text()
		if gr.Match(r, []byte(l)) {
			if _, err := fmt.Fprintln(out, l); err != nil {
				return err
			}
		}
	}
	if err := lines.Err(); err != nil {
		return fmt.Errorf("reading input: %w", err)
	}
	return nil
}
