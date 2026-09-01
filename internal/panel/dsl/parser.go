package dsl

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type tokenKind uint8

const (
	tEOF tokenKind = iota
	tIdent
	tString
	tNumber
	tLBrace
	tRBrace
	tLBracket
	tRBracket
	tEqual
	tComma
)

type token struct {
	kind tokenKind
	lit  string
	pos  int
}

type Block struct {
	Type   string
	Labels []string
	Attrs  map[string]any
	Blocks []Block
}

func Parse(src []byte) ([]Block, error) {
	toks, err := lex(string(src))
	if err != nil {
		return nil, err
	}
	p := parser{tokens: toks}
	var out []Block
	for p.peek().kind != tEOF {
		b, err := p.block()
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

type parser struct {
	tokens []token
	i      int
}

func (p *parser) peek() token { return p.tokens[p.i] }
func (p *parser) next() token { t := p.tokens[p.i]; p.i++; return t }
func (p *parser) expect(k tokenKind) (token, error) {
	t := p.next()
	if t.kind != k {
		return token{}, fmt.Errorf("dsl:%d: expected %v, got %q", t.pos, k, t.lit)
	}
	return t, nil
}
func (p *parser) block() (Block, error) {
	t, err := p.expect(tIdent)
	if err != nil {
		return Block{}, err
	}
	b := Block{Type: t.lit, Attrs: map[string]any{}}
	for p.peek().kind == tString {
		b.Labels = append(b.Labels, p.next().lit)
	}
	if _, err = p.expect(tLBrace); err != nil {
		return Block{}, err
	}
	for p.peek().kind != tRBrace {
		if p.peek().kind == tEOF {
			return Block{}, fmt.Errorf("dsl: unterminated %s block", b.Type)
		}
		name, err := p.expect(tIdent)
		if err != nil {
			return Block{}, err
		}
		if p.peek().kind == tEqual {
			p.next()
			v, err := p.value()
			if err != nil {
				return Block{}, err
			}
			b.Attrs[name.lit] = v
			continue
		}
		child := Block{Type: name.lit, Attrs: map[string]any{}}
		for p.peek().kind == tString {
			child.Labels = append(child.Labels, p.next().lit)
		}
		if _, err := p.expect(tLBrace); err != nil {
			return Block{}, err
		}
		parsed, err := p.blockBody(child)
		if err != nil {
			return Block{}, err
		}
		b.Blocks = append(b.Blocks, parsed)
	}
	p.next()
	return b, nil
}
func (p *parser) blockBody(b Block) (Block, error) {
	for p.peek().kind != tRBrace {
		if p.peek().kind == tEOF {
			return Block{}, fmt.Errorf("dsl: unterminated %s block", b.Type)
		}
		name, err := p.expect(tIdent)
		if err != nil {
			return Block{}, err
		}
		if p.peek().kind == tEqual {
			p.next()
			v, err := p.value()
			if err != nil {
				return Block{}, err
			}
			b.Attrs[name.lit] = v
			continue
		}
		child := Block{Type: name.lit, Attrs: map[string]any{}}
		for p.peek().kind == tString {
			child.Labels = append(child.Labels, p.next().lit)
		}
		if _, err := p.expect(tLBrace); err != nil {
			return Block{}, err
		}
		child, err = p.blockBody(child)
		if err != nil {
			return Block{}, err
		}
		b.Blocks = append(b.Blocks, child)
	}
	p.next()
	return b, nil
}
func (p *parser) value() (any, error) {
	t := p.next()
	switch t.kind {
	case tString:
		return t.lit, nil
	case tNumber:
		if strings.Contains(t.lit, ".") {
			v, e := strconv.ParseFloat(t.lit, 64)
			return v, e
		}
		v, e := strconv.Atoi(t.lit)
		return v, e
	case tIdent:
		if t.lit == "true" {
			return true, nil
		}
		if t.lit == "false" {
			return false, nil
		}
		return nil, fmt.Errorf("dsl:%d: bare value %q is not allowed", t.pos, t.lit)
	case tLBracket:
		var arr []any
		for p.peek().kind != tRBracket {
			v, e := p.value()
			if e != nil {
				return nil, e
			}
			arr = append(arr, v)
			if p.peek().kind == tComma {
				p.next()
			}
		}
		p.next()
		return arr, nil
	default:
		return nil, fmt.Errorf("dsl:%d: invalid value %q", t.pos, t.lit)
	}
}

func lex(s string) ([]token, error) {
	var out []token
	for i := 0; i < len(s); {
		r := rune(s[i])
		if unicode.IsSpace(r) {
			i++
			continue
		}
		if s[i] == '#' {
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}
		if i+1 < len(s) && s[i:i+2] == "//" {
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}
		switch s[i] {
		case '{':
			out = append(out, token{tLBrace, "{", i})
			i++
		case '}':
			out = append(out, token{tRBrace, "}", i})
			i++
		case '[':
			out = append(out, token{tLBracket, "[", i})
			i++
		case ']':
			out = append(out, token{tRBracket, "]", i})
			i++
		case '=':
			out = append(out, token{tEqual, "=", i})
			i++
		case ',':
			out = append(out, token{tComma, ",", i})
			i++
		case '"':
			start := i
			i++
			esc := false
			for i < len(s) {
				if !esc && s[i] == '"' {
					i++
					break
				}
				if !esc && s[i] == '\\' {
					esc = true
					i++
					continue
				}
				esc = false
				i++
			}
			if i > len(s) || s[i-1] != '"' {
				return nil, fmt.Errorf("dsl:%d: unterminated string", start)
			}
			v, err := strconv.Unquote(s[start:i])
			if err != nil {
				return nil, fmt.Errorf("dsl:%d: %w", start, err)
			}
			out = append(out, token{tString, v, start})
		default:
			start := i
			if (s[i] >= '0' && s[i] <= '9') || s[i] == '-' {
				i++
				for i < len(s) && ((s[i] >= '0' && s[i] <= '9') || s[i] == '.') {
					i++
				}
				out = append(out, token{tNumber, s[start:i], start})
				continue
			}
			if isIdentStart(s[i]) {
				i++
				for i < len(s) && isIdentPart(s[i]) {
					i++
				}
				out = append(out, token{tIdent, s[start:i], start})
				continue
			}
			return nil, fmt.Errorf("dsl:%d: unexpected character %q", i, s[i])
		}
	}
	out = append(out, token{kind: tEOF, pos: len(s)})
	return out, nil
}
func isIdentStart(b byte) bool { return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') }
func isIdentPart(b byte) bool {
	return isIdentStart(b) || (b >= '0' && b <= '9') || b == '-' || b == '.'
}
