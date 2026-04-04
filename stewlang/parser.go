package stewlang

import (
	"fmt"
	"strings"
)

type Parser struct {
	tokens []Token
	pos    int
}

func NewParser(tokens []Token) *Parser {
	return &Parser{tokens: tokens, pos: 0}
}

func (p *Parser) Parse() ([]Node, error) {
	return p.parseLoop(TOKEN_EOF, "")
}

func (p *Parser) parseLoop(endToken TokenType, ExpectedComponentName string) ([]Node, error) {
	var nodes []Node

	for p.pos < len(p.tokens) {
		t := p.tokens[p.pos]

		// Check if we hit the terminator for this block loop
		if t.Type == endToken {
			if endToken == TOKEN_COMPONENT_CLOSE {
				name := extractTagNameFromClose(t.Value)
				if name != ExpectedComponentName {
					return nil, fmt.Errorf("line %d: expected closing tag </%s>, found %s", t.Line, ExpectedComponentName, t.Value)
				}
			}
			return nodes, nil
		}

		if t.Type == TOKEN_EOF {
			break
		}
		
		// If we encounter anomalous tokens like ELSE or END that aren't the endToken, it's an error.
		if t.Type == TOKEN_ELSE || t.Type == TOKEN_END || t.Type == TOKEN_COMPONENT_CLOSE {
			return nil, fmt.Errorf("line %d: unexpected token %s", t.Line, t.Type)
		}

		switch t.Type {
		case TOKEN_HTML:
			nodes = append(nodes, NodeHTML{Content: t.Value})
			p.pos++

		case TOKEN_GOSCRIPT_SERVER:
			nodes = append(nodes, NodeGoScript{Content: t.Value, Context: "server"})
			p.pos++

		case TOKEN_GOSCRIPT_CLIENT:
			nodes = append(nodes, NodeGoScript{Content: t.Value, Context: "client"})
			p.pos++

		case TOKEN_BIND:
			nodes = append(nodes, NodeBind{BindType: t.BindType, BindVar: t.BindVar, IsEvent: t.IsEvent})
			p.pos++

		case TOKEN_EXPRESSION:
			nodes = append(nodes, NodeExpression{Content: t.Value})
			p.pos++

		case TOKEN_IF:
			condition := t.Value
			p.pos++

			body, elseBody, err := p.parseIfBody()
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, NodeIf{Condition: condition, Body: body, ElseBody: elseBody})

		case TOKEN_EACH:
			iterator := t.Value
			startLine := t.Line
			p.pos++
			
			body, err := p.parseLoop(TOKEN_END, "")
			if err != nil {
				return nil, err
			}
			if p.pos >= len(p.tokens) || p.tokens[p.pos].Type != TOKEN_END {
				return nil, fmt.Errorf("line %d: missing {{ end }} for each loop", startLine)
			}
			p.pos++ // consume END
			nodes = append(nodes, NodeEach{Iterator: iterator, Body: body})
			
		case TOKEN_COMPONENT_SELF_CLOSING:
			tagName := extractTagNameFromOpen(t.Value)
			nodes = append(nodes, NodeComponent{TagName: tagName, TagContent: t.Value, SelfClosing: true})
			p.pos++
			
		case TOKEN_COMPONENT_OPEN:
			tagStr := t.Value
			tagName := extractTagNameFromOpen(tagStr)
			startLine := t.Line
			p.pos++
			
			body, err := p.parseLoop(TOKEN_COMPONENT_CLOSE, tagName)
			if err != nil {
				return nil, err
			}
			if p.pos >= len(p.tokens) || p.tokens[p.pos].Type != TOKEN_COMPONENT_CLOSE {
				return nil, fmt.Errorf("line %d: missing closing tag </%s>", startLine, tagName)
			}
			p.pos++ // consume CLOSE
			nodes = append(nodes, NodeComponent{TagName: tagName, TagContent: tagStr, SelfClosing: false, Body: body})
			
		case TOKEN_SLOT:
			nodes = append(nodes, NodeSlot{TagContent: t.Value})
			p.pos++
		}
	}

	if endToken != TOKEN_EOF {
		return nil, fmt.Errorf("missing closing token %s", endToken)
	}

	return nodes, nil
}

func (p *Parser) parseIfBody() (body []Node, elseBody []Node, err error) {
	for p.pos < len(p.tokens) {
		t := p.tokens[p.pos]
		if t.Type == TOKEN_EOF {
			return nil, nil, fmt.Errorf("missing {{ end }} for if block")
		}
		if t.Type == TOKEN_END {
			p.pos++ // consume END
			return body, elseBody, nil
		}
		if t.Type == TOKEN_ELSE {
			p.pos++ // consume ELSE
			elseBody, err = p.parseLoop(TOKEN_END, "")
			if err != nil {
				return nil, nil, err
			}
			if p.pos >= len(p.tokens) || p.tokens[p.pos].Type != TOKEN_END {
				return nil, nil, fmt.Errorf("missing {{ end }} for if/else block")
			}
			p.pos++ // consume END
			return body, elseBody, nil
		}
		
		// recursively call parseLoop but we need a way to stop at END *or* ELSE
		// So we parse a single token/construct and append
		node, err := p.parseNode()
		if err != nil {
			return nil, nil, err
		}
		if node != nil {
			body = append(body, node)
		}
	}
	return nil, nil, fmt.Errorf("missing {{ end }} for if block")
}

// parseNode parses exactly one node at the current pos, handling its internal structure
func (p *Parser) parseNode() (Node, error) {
	t := p.tokens[p.pos]
	
	switch t.Type {
	case TOKEN_HTML:
		p.pos++
		return NodeHTML{Content: t.Value}, nil
	case TOKEN_GOSCRIPT_SERVER:
		p.pos++
		return NodeGoScript{Content: t.Value, Context: "server"}, nil
	case TOKEN_GOSCRIPT_CLIENT:
		p.pos++
		return NodeGoScript{Content: t.Value, Context: "client"}, nil
	case TOKEN_BIND:
		p.pos++
		return NodeBind{BindType: t.BindType, BindVar: t.BindVar, IsEvent: t.IsEvent}, nil
	case TOKEN_EXPRESSION:
		p.pos++
		return NodeExpression{Content: t.Value}, nil
	case TOKEN_IF:
		condition := t.Value
		p.pos++
		body, elseBody, err := p.parseIfBody()
		if err != nil {
			return nil, err
		}
		return NodeIf{Condition: condition, Body: body, ElseBody: elseBody}, nil
	case TOKEN_EACH:
		iterator := t.Value
		startLine := t.Line
		p.pos++
		body, err := p.parseLoop(TOKEN_END, "")
		if err != nil {
			return nil, err
		}
		if p.pos >= len(p.tokens) || p.tokens[p.pos].Type != TOKEN_END {
			return nil, fmt.Errorf("line %d: missing {{ end }} for each block", startLine)
		}
		p.pos++ // consume END
		return NodeEach{Iterator: iterator, Body: body}, nil
	case TOKEN_COMPONENT_SELF_CLOSING:
		tagName := extractTagNameFromOpen(t.Value)
		p.pos++
		return NodeComponent{TagName: tagName, TagContent: t.Value, SelfClosing: true}, nil
	case TOKEN_COMPONENT_OPEN:
		tagStr := t.Value
		tagName := extractTagNameFromOpen(tagStr)
		startLine := t.Line
		p.pos++
		body, err := p.parseLoop(TOKEN_COMPONENT_CLOSE, tagName)
		if err != nil {
			return nil, err
		}
		if p.pos >= len(p.tokens) || p.tokens[p.pos].Type != TOKEN_COMPONENT_CLOSE {
			return nil, fmt.Errorf("line %d: missing closing tag </%s>", startLine, tagName)
		}
		p.pos++ // consume CLOSE
		return NodeComponent{TagName: tagName, TagContent: tagStr, SelfClosing: false, Body: body}, nil
	case TOKEN_SLOT:
		p.pos++
		return NodeSlot{TagContent: t.Value}, nil
	default:
		return nil, fmt.Errorf("line %d: unexpected token %v", t.Line, t.Type)
	}
}

func extractTagNameFromOpen(s string) string {
	s = strings.TrimPrefix(s, "<")
	idx := strings.IndexAny(s, " \t\n\r/>")
	if idx == -1 {
		return s
	}
	return s[:idx]
}

func extractTagNameFromClose(s string) string {
	s = strings.TrimPrefix(s, "</")
	s = strings.TrimSuffix(s, ">")
	return strings.TrimSpace(s)
}
