package v1

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

type deviceFilter func(deviceInventory) bool

// Tokens are ASCII identifiers, punctuation, or OData strings with doubled apostrophes.
var deviceFilterToken = regexp.MustCompile(`^(?:[A-Za-z][A-Za-z0-9]*|[(),]|'(?:[^']|'')*')`)

type deviceFilterParser struct {
	rest   string
	token  string
	spaced bool
}

func (p *deviceFilterParser) next() {
	trimmed := strings.TrimLeftFunc(p.rest, unicode.IsSpace)
	p.spaced = len(trimmed) < len(p.rest)
	p.rest = trimmed
	p.token = deviceFilterToken.FindString(p.rest)
	p.rest = p.rest[len(p.token):]
}

func (p *deviceFilterParser) take(token string) bool {
	if p.token != token {
		return false
	}

	p.next()

	return true
}

func parseDeviceFilter(value string) (deviceFilter, error) {
	parser := deviceFilterParser{rest: value}
	parser.next()

	filter, err := parser.expression(0, "or")
	if err != nil {
		return nil, err
	}

	if parser.token != "" || parser.rest != "" {
		return nil, deviceQueryError("unexpected input after expression")
	}

	return filter, nil
}

// An OR expression contains AND expressions, which contain predicates.
func (p *deviceFilterParser) expression(depth int, operator string) (deviceFilter, error) {
	read := func() (deviceFilter, error) { return p.predicate(depth) }
	if operator == "or" {
		read = func() (deviceFilter, error) { return p.expression(depth, "and") }
	}

	first, err := read()
	if err != nil || p.token != operator {
		return first, err
	}

	filters := []deviceFilter{first}

	for p.token == operator {
		if !p.spaced {
			return nil, deviceQueryError(fmt.Sprintf("expected whitespace before %s", operator))
		}

		p.next()

		if !p.spaced {
			return nil, deviceQueryError(fmt.Sprintf("expected whitespace after %s", operator))
		}

		filter, err := read()
		if err != nil {
			return nil, err
		}

		filters = append(filters, filter)
	}

	return func(inventory deviceInventory) bool {
		if operator == "or" {
			return slices.ContainsFunc(filters, func(filter deviceFilter) bool { return filter(inventory) })
		}

		return !slices.ContainsFunc(filters, func(filter deviceFilter) bool { return !filter(inventory) })
	}, nil
}

func (p *deviceFilterParser) predicate(depth int) (deviceFilter, error) {
	const maxGroupDepth = 64
	if p.take("(") {
		if depth >= maxGroupDepth {
			return nil, deviceQueryError("parentheses nested too deeply")
		}

		filter, err := p.expression(depth+1, "or")
		if err != nil {
			return nil, err
		}

		if !p.take(")") {
			return nil, deviceQueryError("expected closing parenthesis")
		}

		return filter, nil
	}

	field := p.token
	p.next()

	switch field {
	case deviceSKUTypeField, deviceCurrentModeField, deviceLMSInstalledField:
		return p.equality(field)
	case "contains":
		return p.textFunction(deviceOSField, func(i deviceInventory, value string) bool {
			return strings.Contains(strings.ToLower(i.os), value)
		})
	case "startswith":
		return p.textFunction(deviceAMTVersionField, func(i deviceInventory, value string) bool {
			return strings.HasPrefix(i.version, value)
		})
	default:
		return nil, deviceQueryError("expected a supported predicate")
	}
}

func (p *deviceFilterParser) text() (string, error) {
	if !strings.HasPrefix(p.token, "'") {
		return "", deviceQueryError("expected a single-quoted string")
	}

	value := strings.ReplaceAll(p.token[1:len(p.token)-1], "''", "'")
	p.next()

	return strings.ToLower(value), nil
}

func (p *deviceFilterParser) textFunction(field string, match func(deviceInventory, string) bool) (deviceFilter, error) {
	if !p.take("(") || !p.take(field) || !p.take(",") {
		return nil, deviceQueryError(fmt.Sprintf("expected %s and a single-quoted string as arguments", field))
	}

	value, err := p.text()
	if err != nil {
		return nil, err
	}

	if !p.take(")") {
		return nil, deviceQueryError("expected closing parenthesis")
	}

	return func(i deviceInventory) bool { return match(i, value) }, nil
}

func (p *deviceFilterParser) equality(field string) (deviceFilter, error) {
	if !p.spaced || !p.take("eq") || !p.spaced {
		return nil, deviceQueryError(fmt.Sprintf("expected eq after %s", field))
	}

	if field == deviceLMSInstalledField {
		value := p.token
		if !p.take("true") && !p.take("false") {
			return nil, deviceQueryError("expected true or false")
		}

		return func(i deviceInventory) bool { return i.lms == value }, nil
	}

	value, err := p.text()
	if err != nil {
		return nil, err
	}

	if field == deviceSKUTypeField {
		if value != "vpro" && value != "ism" && value != deviceSKUTypeNone {
			return nil, deviceQueryError("skuType must be vPro, ISM, or none")
		}

		return func(i deviceInventory) bool { return strings.EqualFold(i.sku, value) }, nil
	}

	if value != "acm" && value != "ccm" && value != "notactivated" {
		return nil, deviceQueryError("currentMode must be ACM, CCM, or notActivated")
	}

	return func(i deviceInventory) bool { return strings.EqualFold(i.mode, value) }, nil
}
