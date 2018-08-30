package query

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	operatorNames = map[string]uint8{
		"==":         Equals,
		">":          GreaterThan,
		">=":         GreaterThanOrEqual,
		"<":          LessThan,
		"<=":         LessThanOrEqual,
		"f==":        FloatEquals,
		"f>":         FloatGreaterThan,
		"f>=":        FloatGreaterThanOrEqual,
		"f<":         FloatLessThan,
		"f<=":        FloatLessThanOrEqual,
		"sameas":     SameAs,
		"s==":        SameAs,
		"contains":   Contains,
		"co":         Contains,
		"startswith": StartsWith,
		"sw":         StartsWith,
		"endswith":   EndsWith,
		"ew":         EndsWith,
		"in":         In,
		"matches":    Matches,
		"re":         Matches,
		"is":         Is,
		"exists":     Exists,
		"ex":         Exists,
	}

	primaryNames = make(map[uint8]string)
)

func init() {
	for opName, opID := range operatorNames {
		name, ok := primaryNames[opID]
		if ok {
			if len(name) < len(opName) {
				primaryNames[opID] = opName
			}
		} else {
			primaryNames[opID] = opName
		}
	}
}

func getOpName(operator uint8) string {
	name, ok := primaryNames[operator]
	if ok {
		return name
	}
	return "[unknown]"
}

type treeElement struct {
	branches       []*treeElement
	text           string
	globalPosition int
}

var (
	escapeReplacer = regexp.MustCompile("\\\\([^\\\\])")
)

// prepToken removes surrounding parenthesis and escape characters.
func prepToken(text string) string {
	return escapeReplacer.ReplaceAllString(strings.Trim(text, "\""), "$1")
}

func extractSnippets(text string) (snippets []*treeElement, err error) {

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

		// wait for parenthesis to be over
		if inParenthesis {
			if char == '"' {
				snippets = append(snippets, &treeElement{
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
				snippets = append(snippets, &treeElement{
					text:           prepToken(text[start:pos]),
					globalPosition: start,
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
			snippets = append(snippets, &treeElement{
				text:           text[pos : pos+1],
				globalPosition: pos,
			})
		case '"':
			if start < pos {
				return nil, fmt.Errorf("parenthesis ('\"') may not be within words, please escape with '\\' (position: %d)", pos+1)
			}
			inParenthesis = true
		}

	}

	// add last
	snippets = append(snippets, &treeElement{
		text:           prepToken(text[start : pos+1]),
		globalPosition: start,
	})

	return snippets, nil

}

// ParseQuery parses a plaintext query. Special characters (that must be escaped with a '\') are: `\()` and any whitespaces.
func ParseQuery(query string) (*Query, error) {
	snippets, err := extractSnippets(query)
	if err != nil {
		return nil, err
	}
	snippetsPos := 0

	getElement := func() (*treeElement, error) {
		if snippetsPos >= len(snippets) {
			return nil, fmt.Errorf("unexpected end at position %d", len(query))
		}
		return snippets[snippetsPos], nil
	}

	// check for query word
	queryWord, err := getElement()
	if err != nil {
		return nil, err
	}
	if queryWord.text != "query" {
		return nil, errors.New("queries must start with \"query\"")
	}

	// get prefix
	prefix, err := getElement()
	if err != nil {
		return nil, err
	}

	// check if no condition
	if len(snippets) == 2 {
		return New(prefix.text, nil)
	}

	// check for where word
	whereWord, err := getElement()
	if err != nil {
		return nil, err
	}
	if whereWord.text != "where" {
		return nil, errors.New("filtering queries must start conditions with \"where\"")
	}

	// parse conditions
	condition, err := parseCondition(getElement)
	if err != nil {
		return nil, err
	}

	// check for additional tokens
	// token := s.Scan()
	// if token != scanner.EOF {
	// 	return nil, fmt.Errorf("unexpected additional tokens at position %d", s.Position)
	// }

	return New(prefix.text, condition)
}

func parseCondition(getElement func() (*treeElement, error)) (Condition, error) {
	first, err := getElement()
	if err != nil {
		return nil, err
	}

	switch first.text {
	case "(":
		return parseAndOr(getElement, true, nil, false)
		// case ""
	}

	return nil, nil
}

func parseAndOr(getElement func() (*treeElement, error), expectBracket bool, preParsedCondition Condition, preParsedIsOr bool) (Condition, error) {
	return nil, nil
}
