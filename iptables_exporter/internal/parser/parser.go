package parser

// parser.go 新增内容
import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var (
	ruleLineRe = regexp.MustCompile(`^\[(\d+):(\d+)\]\s+(-A\s+\S+.*)`)
)

// GetTables 对外暴露的获取表数据方法context
// 修改 GetTables 函数
func GetTables(ctx context.Context) (Tables, error) {
	cmd := exec.CommandContext(ctx, "iptables-save", "-c")
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start iptables-save: %w", err)
	}

	var wg sync.WaitGroup
	var tables Tables
	var parseErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		tables, parseErr = ParseIptablesSave(pipe)
		// 显式关闭管道（重要）
		if closer, ok := pipe.(io.Closer); ok {
			_ = closer.Close()
		}
	}()

	// 先等待解析完成
	wg.Wait()

	// 再等待命令退出
	if err := cmd.Wait(); err != nil {
		return tables, fmt.Errorf("iptables-save execution failed (partial data may exist): %w", err)
	}

	return tables, parseErr
}

func ParseIptablesSave(r io.Reader) (Tables, error) {
	reader := bufio.NewReader(r)
	var parser parser
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		parser.handleLine(strings.TrimSpace(line)) // 去除前后空白
	}
	parser.flush()
	if len(parser.errors) > 0 {
		return nil, parser.errors[0] // 只返回第一个错误
	}
	return parser.result, nil
}

func (e ParseError) Error() string {
	return fmt.Sprintf("Error in table '%s', chain '%s' at line %d: %s (content: %+v)",
		e.Table, e.Chain, e.LineNumber, e.Message, e.LineText)
}

func (p *parser) flush() {
	if p.currentTableName == "" {
		return
	}

	if p.currentTable != nil {
		for _, chain := range p.currentTable.Chains {
			if chain.Policy == "" { // 示例验证逻辑
				p.errors = append(p.errors, ParseError{
					Message:  "chain policy missing",
					LineText: fmt.Sprintf(":%s", chain.Name),
				})
			}
		}
	}

	if p.result == nil {
		p.result = make(Tables)
	}
	// 确保指针不为nil时设置Name
	if p.currentTable != nil {
		p.currentTable.Name = p.currentTableName
		p.result[p.currentTableName] = p.currentTable
	}
	p.currentTableName = ""
	p.currentTable = nil // 现在可以正确赋nil
}

func (p *parser) handleNewChain(line string) {
	fields := strings.Fields(line)
	if len(fields) != 3 {
		p.errors = append(p.errors, ParseError{"expected 3 fields", p.line, line, p.currentTableName, ""})
		return
	}

	name := strings.TrimPrefix(fields[0], ":")
	policy := fields[1]
	if policy != "ACCEPT" && policy != "DROP" && policy != "REJECT" && policy != "-" {
		p.errors = append(p.errors, ParseError{"invalid policy", p.line, line, p.currentTableName, name})
		return
	}

	packets, bytes, ok := parseCounters(fields[2])
	if !ok {
		p.errors = append(p.errors, ParseError{"expected [packets:bytes]", p.line, line, p.currentTableName, name})
		return
	}

	if p.currentTable == nil {
		p.currentTable = &Table{
			Name:   p.currentTableName,
			Chains: map[string]Chain{},
		}
	}
	p.currentTable.Chains[name] = Chain{
		Name:    name,
		Policy:  policy,
		Packets: packets,
		Bytes:   bytes,
		Rules:   []Rule{},
	}
}

func parseCounters(token string) (uint64, uint64, bool) {
	token = strings.Trim(token, "[]")
	if token == "" { // 新增空值检查
		return 0, 0, false
	}

	parts := strings.Split(token, ":") // 改为冒号分割
	if len(parts) != 2 {
		return 0, 0, false
	}

	// 处理可能存在的空格
	parts[0] = strings.TrimSpace(parts[0])
	parts[1] = strings.TrimSpace(parts[1])

	var packets, bytes uint64
	_, err1 := fmt.Sscanf(parts[0], "%d", &packets)
	_, err2 := fmt.Sscanf(parts[1], "%d", &bytes)
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return packets, bytes, true
}

func (p *parser) handleLine(line string) {
	p.line++
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return
	}

	switch {
	case line == "COMMIT":
		p.flush()
	case strings.HasPrefix(line, "*"):
		p.flush()
		p.currentTableName = strings.TrimPrefix(line, "*")
	case strings.HasPrefix(line, ":"):
		p.handleNewChain(line)
	default:
		// 尝试匹配带计数器的规则行
		if matches := ruleLineRe.FindStringSubmatch(line); matches != nil {
			// 提取计数器并处理规则
			packets, _ := strconv.ParseUint(matches[1], 10, 64)
			bytes, _ := strconv.ParseUint(matches[2], 10, 64)
			p.handleRuleWithCounter(matches[3], packets, bytes)
		} else if strings.HasPrefix(line, "-A") {
			p.handleRule(line)
		} else {
			p.errors = append(p.errors, ParseError{"unhandled line", p.line, line, p.currentTableName, ""})
		}
	}
}

func (p *parser) handleRule(line string) {
	fields := strings.Fields(line)
	var subParser ruleParser
	for _, token := range fields {
		subParser.handleToken(token)
	}
	subParser.flush()

	if !subParser.countersOk || subParser.chain == "" {
		p.errors = append(p.errors, ParseError{"malformed rule", p.line, line, p.currentTableName, subParser.chain})
		return
	}

	r := Rule{
		Packets:  subParser.packets,
		Bytes:    subParser.bytes,
		RuleSpec: strings.TrimPrefix(strings.Join(subParser.flags, " "), "-A"),
	}
	chain := p.currentTable.Chains[subParser.chain]
	chain.Rules = append(chain.Rules, r)
	p.currentTable.Chains[subParser.chain] = chain
}

// 新增带计数器的规则处理方法
func (p *parser) handleRuleWithCounter(ruleLine string, packets, bytes uint64) {
	fields := strings.Fields(ruleLine)
	var subParser ruleParser
	subParser.packets = packets // 注入已解析的计数器
	subParser.bytes = bytes
	subParser.countersOk = true // 标记计数器已解析

	for _, token := range fields {
		subParser.handleToken(token)
	}
	subParser.flush()

	if subParser.chain == "" {
		p.errors = append(p.errors, ParseError{"malformed rule", p.line, ruleLine, p.currentTableName, subParser.chain})
		return
	}

	r := Rule{
		Packets:  subParser.packets,
		Bytes:    subParser.bytes,
		RuleSpec: strings.TrimPrefix(strings.Join(subParser.flags, " "), "-A"),
	}
	chain := p.currentTable.Chains[subParser.chain]
	chain.Rules = append(chain.Rules, r)
	p.currentTable.Chains[subParser.chain] = chain
}
