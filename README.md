# Generalized Regular Expression

This repository provides an implementation of generalized regular
expressions following the [Brzozowski
derivative](https://en.wikipedia.org/wiki/Brzozowski_derivative)
method of matching. It contains a simple package to compute the
derivative of a regexp, a `filter` package that reads a generalized
regular expression written in an easy to parse postfix notation and
selects the lines of a text matching it, and a command line tool
applying the latter to `stdin`.

A generalized regular expression adds the intersection (`and`) and
complement (`not`) operations to the usual regular expressions.

## Building

The repository builds with the Go tool and with Bazel. Both read the
Go version and the dependency list from `go.mod`; with Bazel that is
done by `MODULE.bazel`, which uses Bzlmod exclusively (there is no
`WORKSPACE` file).

    go build ./... && go test ./...
    bazel build //... && bazel test //...

## Command line tool

The tool reads `stdin` and writes back the lines matched by the
expression given on the command line:

    go build -o gr ./cmd
    gr <expression> < input.txt

Under Bazel the same binary is `//cmd`:

    bazel run //cmd -- <expression> < input.txt

A line is printed when the expression matches it *entirely*; there is
no implicit search. Use the `,contains` operator (or surround the
pattern with `,any`) to match a substring instead.

### Postfix notation

The expression is a sequence of arguments in postfix (reverse Polish)
notation, evaluated against a stack. An argument starting with a comma
is an operator, and pops its operands off the stack; anything else is
a literal string and is pushed onto the stack. Once every argument has
been consumed, exactly one expression must be left on the stack.

For example, `gr foo bar ,or` pushes the literals `foo` and `bar`,
then `,or` pops both and pushes their union, leaving one expression:
lines that read exactly `foo` or exactly `bar`.

| Operator    | Operands | Meaning                                       |
|-------------|----------|-----------------------------------------------|
| `,concat`   | 2        | the two operands in sequence, `ab`             |
| `,or`       | 2        | union, `a\|b`                                  |
| `,and`      | 2        | intersection: matches iff both operands match  |
| `,not`      | 1        | complement: matches iff the operand does not   |
| `,star`     | 1        | zero or more repetitions, `a*`                 |
| `,plus`     | 1        | one or more repetitions, `a+`                  |
| `,contains` | 1        | the operand anywhere in the line, `.*a.*`      |
| `,dot`      | 0        | any single rune, `.`                           |
| `,any`      | 0        | any string, including the empty one, `.*`      |
| `,empty`    | 0        | the empty set: matches nothing at all          |

Binary operators take their operands in the order they were pushed, so
`gr a b ,concat` matches `ab`, not `ba`.

Matching works on runes, not bytes: `,dot` matches one `é` as a single
character.

### Quoting

To match a literal starting with a comma, double it: `,,star` is the
literal string `,star`, whereas `,star` is the repetition operator.
Only the first comma is removed.

The empty argument `""` is the literal empty string, which matches
only the empty line.

### Examples

Lines containing `foo` but not `bar` — the combination that plain
regular expressions cannot express directly:

    gr foo ,contains bar ,contains ,not ,and

Lines consisting of one or more repetitions of `ab`:

    gr ab ,plus

Lines containing both `foo` and `bar`, in either order:

    gr foo ,contains bar ,contains ,and

Non-empty lines that do not contain a space:

    gr ,dot ,plus " " ,contains ,not ,and

### Exit status

The tool exits with `0` after reading all of its input, and with `1`
if the expression is malformed, after printing the reason on `stdout`:

    $ gr a b
    invalid length in stack: ["a" "b"]
    $ gr ,concat
    not enough arguments to "concat"
    $ gr ,bogus
    unknown operator ",bogus" (use ",," to escape a leading comma)

### Limitations

Input lines are read with a `bufio.Scanner` left at its default buffer
size, so a line longer than 64 KiB cannot be matched. Such a line is
reported rather than skipped, and the lines that follow it are not
read at all:

    $ gr foo ,contains < very-long-lines.txt
    reading input: bufio.Scanner: token too long
