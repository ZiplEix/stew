package stewlang

type Node interface {
	nodeType() string
}

type NodeHTML struct {
	Content string
}

func (n NodeHTML) nodeType() string { return "HTML" }

type NodeGoScript struct {
	Content string
	Context string // "server" or "client"
}

func (n NodeGoScript) nodeType() string { return "GOSCRIPT" }

type NodeExpression struct {
	Content string
}

func (n NodeExpression) nodeType() string { return "EXPRESSION" }

type NodeIf struct {
	Condition string
	Body      []Node
	ElseBody  []Node
	BlockID   string // Generated for reactivity
}

func (n NodeIf) nodeType() string { return "IF" }

type NodeEach struct {
	Iterator string
	Body     []Node
	BlockID  string // Generated for reactivity
}

func (n NodeEach) nodeType() string { return "EACH" }

type NodeComponent struct {
	TagName     string
	TagContent  string // full string like `<Button title="Hello" />`
	SelfClosing bool
	Body        []Node // populated if not self-closing
}

func (n NodeComponent) nodeType() string { return "COMPONENT" }

type NodeSlot struct {
	TagContent string
}

func (n NodeSlot) nodeType() string { return "SLOT" }

type NodeBind struct {
	BindType string // e.g. "value", "content"
	BindVar  string // e.g. "name"
	IsEvent  bool
}

func (n NodeBind) nodeType() string { return "BIND" }
