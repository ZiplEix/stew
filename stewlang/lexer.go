package stewlang

import (
	"regexp"
	"strings"
	"unicode"
)

type Lexer struct {
	input           string
	pos             int
	line            int
	ValidComponents map[string]string
	moduleBase      string
	relFilePath     string
	insideTag       bool
}

func NewLexer(input string, moduleBase string, relFilePath string) *Lexer {
	l := &Lexer{
		input:           input,
		pos:             0,
		line:            1,
		ValidComponents: make(map[string]string),
		moduleBase:      moduleBase,
		relFilePath:     relFilePath,
		insideTag:       false,
	}

	// First pass parsing for Component Imports
	// e.g., import "./components/Button.stew" -> registers "Button"
	re := regexp.MustCompile(`import\s+["']([^"']*\.stew)["']`)
	for _, match := range re.FindAllStringSubmatch(input, -1) {
		importPath := match[1]
		
		// extract component name
		compName := importPath
		lastSlash := strings.LastIndex(compName, "/")
		if lastSlash != -1 {
			compName = compName[lastSlash+1:]
		}
		compName = strings.TrimSuffix(compName, ".stew")
		
		// If same directory, no prefix
		if strings.HasPrefix(importPath, "./") && strings.Count(importPath, "/") == 1 {
			l.ValidComponents[compName] = ""
		} else {
			dir := importPath[:lastSlash] // e.g. "./components"
			alias := dir
			lastAliasSlash := strings.LastIndex(dir, "/")
			if lastAliasSlash != -1 {
				alias = dir[lastAliasSlash+1:]
			} else {
				alias = strings.TrimPrefix(alias, "./")
				alias = strings.TrimPrefix(alias, "../")
			}
			l.ValidComponents[compName] = alias + "."
		}
	}

	return l
}

func (l *Lexer) Lex() []Token {
	var tokens []Token

	for l.pos < len(l.input) {
		if strings.HasPrefix(l.input[l.pos:], "{{") {
			tokens = append(tokens, l.lexExpressionGroup())
			continue
		}

		if strings.HasPrefix(l.input[l.pos:], "<goscript") && l.isTopLevel() {
			tokens = append(tokens, l.lexGoScript())
			continue
		}

		// Only match bindings inside a tag
		if l.insideTag && (strings.HasPrefix(l.input[l.pos:], "bind:") || strings.HasPrefix(l.input[l.pos:], "on:")) {
			tokens = append(tokens, l.lexBindAttribute())
			continue
		}

		if l.isComponentStart() || l.isComponentClose() || l.isSlot() {
			l.insideTag = true // Components start a tag context
			tokens = append(tokens, l.lexComponentOrSlot())
			// Component/Slot lexing consumes the whole tag including '>'
			l.insideTag = false
			continue
		}

		tokens = append(tokens, l.lexHTML())
		
		// After lexHTML returns, if the last consumed character was '>', set insideTag to false
		// This handles standard HTML tags like <div>
		if l.pos > 0 && l.input[l.pos-1] == '>' {
			l.insideTag = false
		}
	}

	tokens = append(tokens, Token{Type: TOKEN_EOF, Line: l.line})
	return tokens
}

func (l *Lexer) isComponentStart() bool {
	if !strings.HasPrefix(l.input[l.pos:], "<") {
		return false
	}
	if l.pos+1 < len(l.input) && unicode.IsUpper(rune(l.input[l.pos+1])) {
		name := extractTagNameQuick(l.input[l.pos+1:])
		_, exists := l.ValidComponents[name]
		return exists
	}
	return false
}

func (l *Lexer) isComponentClose() bool {
	if !strings.HasPrefix(l.input[l.pos:], "</") {
		return false
	}
	if l.pos+2 < len(l.input) && unicode.IsUpper(rune(l.input[l.pos+2])) {
		name := extractTagNameQuick(l.input[l.pos+2:])
		_, exists := l.ValidComponents[name]
		return exists
	}
	return false
}

func extractTagNameQuick(s string) string {
	idx := strings.IndexAny(s, " \t\n\r/>")
	if idx == -1 {
		return s
	}
	return s[:idx]
}

func (l *Lexer) isSlot() bool {
	return strings.HasPrefix(l.input[l.pos:], "<slot")
}

func (l *Lexer) lexExpressionGroup() Token {
	startLine := l.line
	l.pos += 2 // skip "{{"
	
	// find closing "}}"
	endPos := strings.Index(l.input[l.pos:], "}}")
	if endPos == -1 {
		// unclosed expression, absorb the rest
		val := l.input[l.pos:]
		l.advance(len(val))
		return Token{Type: TOKEN_EXPRESSION, Value: strings.TrimSpace(val), Line: startLine}
	}
	
	val := l.input[l.pos : l.pos+endPos]
	l.advance(endPos + 2) // skip "}}"
	
	valTrimmed := strings.TrimSpace(val)
	
	if strings.HasPrefix(valTrimmed, "if ") {
		return Token{Type: TOKEN_IF, Value: strings.TrimSpace(valTrimmed[3:]), Line: startLine}
	}
	if valTrimmed == "else" {
		return Token{Type: TOKEN_ELSE, Value: "", Line: startLine}
	}
	if strings.HasPrefix(valTrimmed, "each ") {
		return Token{Type: TOKEN_EACH, Value: strings.TrimSpace(valTrimmed[5:]), Line: startLine}
	}
	if valTrimmed == "end" {
		return Token{Type: TOKEN_END, Value: "", Line: startLine}
	}
	
	return Token{Type: TOKEN_EXPRESSION, Value: valTrimmed, Line: startLine}
}

func (l *Lexer) lexGoScript() Token {
	startLine := l.line
	
	endTagStart := strings.Index(l.input[l.pos:], ">") 
	if endTagStart == -1 {
		val := l.input[l.pos:]
		l.advance(len(val))
		return Token{Type: TOKEN_GOSCRIPT_SERVER, Value: val, Line: startLine}
	}
	
	tagFull := l.input[l.pos : l.pos+endTagStart+1]
	tokenType := TOKEN_GOSCRIPT_SERVER
	if strings.Contains(tagFull, "client") {
		tokenType = TOKEN_GOSCRIPT_CLIENT
	}
	
	l.advance(endTagStart + 1) // skip "<goscript ...>"
	
	endPos := strings.Index(l.input[l.pos:], "</goscript>")
	if endPos == -1 {
		val := l.input[l.pos:]
		l.advance(len(val))
		return Token{Type: tokenType, Value: val, Line: startLine}
	}
	
	val := l.input[l.pos : l.pos+endPos]
	l.advance(endPos + 11) // skip "</goscript>"
	return Token{Type: tokenType, Value: val, Line: startLine}
}

func (l *Lexer) lexBindAttribute() Token {
	startLine := l.line
	input := l.input[l.pos:]

	prefix := ""
	if strings.HasPrefix(input, "bind:") {
		prefix = "bind"
	} else if strings.HasPrefix(input, "on:") {
		prefix = "on"
	} else {
		return l.lexHTML()
	}

	// Find the '=' after the prefix
	eqIdx := strings.Index(input, "=")
	if eqIdx == -1 {
		l.advance(len(prefix) + 1)
		return Token{Type: TOKEN_HTML, Value: prefix, Line: startLine}
	}

	attrName := input[len(prefix)+1 : eqIdx]
	l.advance(eqIdx + 1) // skip prefix:name=
	input = l.input[l.pos:]

	var bindVar string
	var fullMatch string

	if strings.HasPrefix(input, "{{") {
		// Balanced braces parsing for {{ ... }}
		l.advance(2)
		braceCount := 1
		startPos := l.pos
		for l.pos < len(l.input)-1 && braceCount > 0 {
			if l.input[l.pos] == '{' && l.input[l.pos+1] == '{' {
				braceCount++
				l.pos += 2
			} else if l.input[l.pos] == '}' && l.input[l.pos+1] == '}' {
				braceCount--
				if braceCount > 0 {
					l.pos += 2
				}
			} else {
				l.pos++
			}
		}
		if braceCount == 0 {
			bindVar = strings.TrimSpace(l.input[startPos : l.pos])
			l.advance(2) // skip closing }}
			fullMatch = prefix + ":" + attrName + "={{" + bindVar + "}}"
		}
	} else if strings.HasPrefix(input, "\"") {
		// Quoted string for literal binding
		l.advance(1)
		quoteEnd := strings.Index(l.input[l.pos:], "\"")
		if quoteEnd != -1 {
			bindVar = "\"" + l.input[l.pos:l.pos+quoteEnd] + "\""
			l.advance(quoteEnd + 1)
			fullMatch = prefix + ":" + attrName + "=" + bindVar
		}
	}

	if fullMatch != "" {
		return Token{
			Type:     TOKEN_BIND,
			Value:    fullMatch,
			BindType: attrName,
			BindVar:  bindVar,
			IsEvent:  prefix == "on",
			Line:     startLine,
		}
	}

	// If it fails to match a valid binding pattern, treat the prefix as HTML
	return Token{Type: TOKEN_HTML, Value: prefix, Line: startLine}
}

func (l *Lexer) lexComponentOrSlot() Token {
	startLine := l.line
	
	// Advance until '>'
	endPos := strings.Index(l.input[l.pos:], ">")
	if endPos == -1 {
		// Syntax error, just consume to end
		val := l.input[l.pos:]
		l.advance(len(val))
		return Token{Type: TOKEN_HTML, Value: val, Line: startLine}
	}
	
	tagStr := l.input[l.pos : l.pos+endPos+1]
	l.advance(endPos + 1)
	
	if strings.HasPrefix(tagStr, "</") {
		return Token{Type: TOKEN_COMPONENT_CLOSE, Value: tagStr, Line: startLine}
	}
	
	if strings.HasPrefix(tagStr, "<slot") {
		return Token{Type: TOKEN_SLOT, Value: tagStr, Line: startLine}
	}
	
	if strings.HasSuffix(tagStr, "/>") {
		return Token{Type: TOKEN_COMPONENT_SELF_CLOSING, Value: tagStr, Line: startLine}
	}
	
	return Token{Type: TOKEN_COMPONENT_OPEN, Value: tagStr, Line: startLine}
}

func (l *Lexer) lexHTML() Token {
	startLine := l.line
	startPos := l.pos
	
	for l.pos < len(l.input) {
		// Break if we see something that needs special Lexing
		if strings.HasPrefix(l.input[l.pos:], "{{") || 
		   (strings.HasPrefix(l.input[l.pos:], "<goscript") && l.isTopLevel()) ||
		   (l.insideTag && (strings.HasPrefix(l.input[l.pos:], "bind:") || strings.HasPrefix(l.input[l.pos:], "on:"))) ||
		   l.isComponentStart() || l.isComponentClose() || l.isSlot() {
			
			// If it might be a binding, we enforce that it MUST be preceded by whitespace
			// to avoid matching <code>bind:</code> or text like (bind:)
			if l.insideTag && (strings.HasPrefix(l.input[l.pos:], "bind:") || strings.HasPrefix(l.input[l.pos:], "on:")) {
				if l.pos > 0 && !unicode.IsSpace(rune(l.input[l.pos-1])) {
					// Not preceded by space, just continue lexing as ordinary HTML
				} else {
					// Good match, break to let Lex() handle it
					break
				}
			} else {
				break
			}
		}
		
		if l.input[l.pos] == '<' && !strings.HasPrefix(l.input[l.pos:], "{{") {
			// If we see a new '<' and we are already at l.pos > startPos, 
			// we should probably break to let the main loop handle the new tag start.
			if l.pos > startPos {
				break
			}
			// Only set insideTag if it looks like a tag start (followed by a letter)
			if l.pos+1 < len(l.input) && unicode.IsLetter(rune(l.input[l.pos+1])) {
				l.insideTag = true
			}
		} else if l.input[l.pos] == '>' {
			l.insideTag = false
			l.pos++ // consume '>'
			break
		}
		
		if l.input[l.pos] == '\n' {
			l.line++
		}
		l.pos++
	}
	
	return Token{Type: TOKEN_HTML, Value: l.input[startPos:l.pos], Line: startLine}
}

func (l *Lexer) isTopLevel() bool {
	if l.pos == 0 {
		return true
	}
	// find start of current line
	start := l.pos
	for start > 0 && l.input[start-1] != '\n' {
		start--
	}
	// check if all characters before l.pos on this line are whitespace
	for i := start; i < l.pos; i++ {
		if !unicode.IsSpace(rune(l.input[i])) {
			return false
		}
	}
	return true
}

func (l *Lexer) advance(n int) {
	str := l.input[l.pos : l.pos+n]
	l.line += strings.Count(str, "\n")
	l.pos += n
}
