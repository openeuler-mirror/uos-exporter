package parser

type Tables map[string]*Table

type Table struct {
	Name   string
	Chains map[string]Chain
}

type Chain struct {
	Name    string
	Policy  string
	Packets uint64
	Bytes   uint64
	Rules   []Rule
}

type Rule struct {
	RuleSpec string
	Packets  uint64
	Bytes    uint64
}

type parser struct {
	result           Tables
	currentTableName string
	currentTable     *Table
	line             int
	errors           []error
}

type ruleParser struct {
	packets       uint64
	bytes         uint64
	countersOk    bool
	current       string
	currentValues []string
	chain         string
	flags         []string
}

type ParseError struct {
	Message    string
	LineNumber int
	LineText   string
	Table      string
	Chain      string
}
