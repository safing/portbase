package query

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type snippet struct {
	text           string
	globalPosition int
}

// ParseQuery parses a plaintext query. Special characters (that must be escaped with a '\') are: `\()` and any whitespaces.
//nolint:gocognit
func ParseQuery(query string) (*Query, error) {
	snippets, err := extractSnippets(query)
	if err != nil {
		return nil, err
	}
	snippetsPos := 0

	getSnippet := func() (*snippet, error) {
		// order is important, as parseAndOr will always consume one additional snippet.
		snippetsPos++
		if snippetsPos > len(snippets) {
			return nil, &SyntaxError{Pos: len(query), Msg: "unexpected end"}
		}
		return snippets[snippetsPos-1], nil
	}
	remainingSnippets := func() int {
		return len(snippets) - snippetsPos
	}

	// check for query word
	queryWord, err := getSnippet()
	if err != nil {
		return nil, err
	}
	if queryWord.text != "query" {
		return nil, syntaxErr(queryWord, "queries must start with \"query\"")
	}

	// get prefix
	prefix, err := getSnippet()
	if err != nil {
		return nil, err
	}
	q := New(prefix.text)

	for remainingSnippets() > 0 {
		command, err := getSnippet()
		if err != nil {
			return nil, err
		}

		switch command.text {
		case "where":
			if q.where != nil {
				return nil, syntaxErr(command, "duplicate clause")
			}

			// parse conditions
			condition, err := parseAndOr(getSnippet, remainingSnippets, true)
			if err != nil {
				return nil, err
			}
			// go one back, as parseAndOr had to check if its done
			snippetsPos--

			q.Where(condition)
		case "orderby":
			if q.orderBy != "" {
				return nil, syntaxErr(command, "duplicate clause")
			}

			orderBySnippet, err := getSnippet()
			if err != nil {
				return nil, err
			}

			q.OrderBy(orderBySnippet.text)
		case "limit":
			if q.limit != 0 {
				return nil, syntaxErr(command, "duplicate clause")
			}

			limitSnippet, err := getSnippet()
			if err != nil {
				return nil, err
			}
			limit, err := strconv.ParseUint(limitSnippet.text, 10, 31)
			if err != nil {
				return nil, syntaxErr(limitSnippet, "invalid integer")
			}

			q.Limit(int(limit))
		case "offset":
			if q.offset != 0 {
				return nil, syntaxErr(command, "duplicate clause")
			}

			offsetSnippet, err := getSnippet()
			if err != nil {
				return nil, err
			}
			offset, err := strconv.ParseUint(offsetSnippet.text, 10, 31)
			if err != nil {
				return nil, syntaxErr(offsetSnippet, "invalid integer")
			}

			q.Offset(int(offset))
		default:
			return nil, syntaxErr(command, "unknown clause")
		}
	}

	return q.Check()
}

func extractSnippets(text string) (snippets []*snippet, err error) {
	skip := false
	start := -1
	inParenthesis := false
	var pos int
	var char rune

	for pos, char = range text {

		// skip
		if skip {
			skip = false
			continue
		}
		if char == '\\' {
			skip = true
		}

		// wait for parenthesis to be overs
		if inParenthesis {
			if char == '"' {
				snippets = append(snippets, &snippet{
					text:           prepToken(text[start+1 : pos]),
					globalPosition: start + 1,
				})
				start = -1
				inParenthesis = false
			}
			continue
		}

		// handle segments
		switch char {
		case '\t', '\n', '\r', ' ', '(', ')':
			if start >= 0 {
				snippets = append(snippets, &snippet{
					text:           prepToken(text[start:pos]),
					globalPosition: start + 1,
				})
				start = -1
			}
		default:
			if start == -1 {
				start = pos
			}
		}

		// handle special segment characters
		switch char {
		case '(', ')':
			snippets = append(snippets, &snippet{
				text:           text[pos : pos+1],
				globalPosition: pos + 1,
			})
		case '"':
			if start < pos {
				return nil, &SyntaxError{Pos: pos + 1, Msg: "parenthesis ('\"') may not be used within words, please escape with '\\'"}
			}
			inParenthesis = true
		}

	}

	// add last
	if start >= 0 {
		snippets = append(snippets, &snippet{
			text:           prepToken(text[start : pos+1]),
			globalPosition: start + 1,
		})
	}

	return snippets, nil
}

//nolint:gocognit
func parseAndOr(getSnippet func() (*snippet, error), remainingSnippets func() int, rootCondition bool) (Condition, error) {
	isOr := false
	typeSet := false
	wrapInNot := false
	expectingMore := true
	var conditions []Condition

	for {
		if !expectingMore && rootCondition && remainingSnippets() == 0 {
			// advance snippetsPos by one, as it will be set back by 1
			getSnippet() //nolint:errcheck
			if len(conditions) == 1 {
				return conditions[0], nil
			}
			if isOr {
				return Or(conditions...), nil
			}
			return And(conditions...), nil
		}

		firstSnippet, err := getSnippet()
		if err != nil {
			return nil, err
		}

		if !expectingMore && rootCondition {
			switch firstSnippet.text {
			case "orderby", "limit", "offset":
				if len(conditions) == 1 {
					return conditions[0], nil
				}
				if isOr {
					return Or(conditions...), nil
				}
				return And(conditions...), nil
			}
		}

		switch firstSnippet.text {
		case "(":
			condition, err := parseAndOr(getSnippet, remainingSnippets, false)
			if err != nil {
				return nil, err
			}
			if wrapInNot {
				conditions = append(conditions, Not(condition))
				wrapInNot = false
			} else {
				conditions = append(conditions, condition)
			}
			expectingMore = true
		case ")":
			if len(conditions) == 1 {
				return conditions[0], nil
			}
			if isOr {
				return Or(conditions...), nil
			}
			return And(conditions...), nil
		case "and":
			if typeSet && isOr {
				return nil, syntaxErr(firstSnippet, "mix \"and\" and \"or\"")
			}
			isOr = false
			typeSet = true
			expectingMore = true
		case "or":
			if typeSet && !isOr {
				return nil, syntaxErr(firstSnippet, "mix \"and\" and \"or\"")
			}
			isOr = true
			typeSet = true
			expectingMore = true
		case "not":
			wrapInNot = true
			expectingMore = true
		default:
			condition, err := parseCondition(firstSnippet, getSnippet)
			if err != nil {
				return nil, err
			}
			if wrapInNot {
				conditions = append(conditions, Not(condition))
				wrapInNot = false
			} else {
				conditions = append(conditions, condition)
			}
			expectingMore = false
		}
	}
}

func parseCondition(firstSnippet *snippet, getSnippet func() (*snippet, error)) (Condition, error) {
	wrapInNot := false

	// get operator name
	opName, err := getSnippet()
	if err != nil {
		return nil, err
	}
	// negate?
	if opName.text == "not" {
		wrapInNot = true
		opName, err = getSnippet()
		if err != nil {
			return nil, err
		}
	}

	// get operator
	operator, ok := operatorNames[opName.text]
	if !ok {
		return nil, syntaxErr(opName, "unknown operator")
	}

	// don't need a value for "exists"
	if operator == Exists {
		if wrapInNot {
			return Not(Where(firstSnippet.text, operator, nil)), nil
		}
		return Where(firstSnippet.text, operator, nil), nil
	}

	// get value
	value, err := getSnippet()
	if err != nil {
		return nil, err
	}
	if wrapInNot {
		return Not(Where(firstSnippet.text, operator, value.text)), nil
	}
	return Where(firstSnippet.text, operator, value.text), nil
}

var escapeReplacer = regexp.MustCompile(`\\([^\\])`)

// prepToken removes surrounding parenthesis and escape characters.
func prepToken(text string) string {
	return escapeReplacer.ReplaceAllString(strings.Trim(text, "\""), "$1")
}

// escapeString correctly escapes a snippet for printing.
func escapeString(token string) string {
	// check if token contains characters that need to be escaped
	if strings.ContainsAny(token, "()\"\\\t\r\n ") {
		// put the token in parenthesis and only escape \ and "
		return fmt.Sprintf("\"%s\"", strings.Replace(token, "\"", "\\\"", -1))
	}
	return token
}
