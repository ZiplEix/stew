package stewlang

type TokenType string

const (
	TOKEN_HTML                   TokenType = "HTML"
	TOKEN_GOSCRIPT               TokenType = "GOSCRIPT"
	TOKEN_EXPRESSION             TokenType = "EXPRESSION" // e.g., {{ data.User.Name }}
	TOKEN_IF                     TokenType = "IF"         // e.g., {{ if ... }}
	TOKEN_ELSE                   TokenType = "ELSE"       // e.g., {{ else }}
	TOKEN_EACH                   TokenType = "EACH"       // e.g., {{ each ... }}
	TOKEN_END                    TokenType = "END"        // e.g., {{ end }}
	TOKEN_EOF                    TokenType = "EOF"
	TOKEN_COMPONENT_OPEN         TokenType = "COMPONENT_OPEN"
	TOKEN_COMPONENT_CLOSE        TokenType = "COMPONENT_CLOSE"
	TOKEN_COMPONENT_SELF_CLOSING TokenType = "COMPONENT_SELF_CLOSING"
	TOKEN_SLOT                   TokenType = "SLOT"
)

type Token struct {
	Type  TokenType
	Value string
	Line  int
}
