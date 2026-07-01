package parser

import (
	"strings"
)

func (p *ruleParser) flush() {
	if p.current == "" {
		return
	}
	if len(p.currentValues) > 0 {
		switch p.current {
		case "-A", "--append":
			// 确保 `-A` 规则赋值 chain
			p.chain = p.currentValues[0]
		default:
			p.flags = append(p.flags, p.current)
			p.flags = append(p.flags, p.currentValues...)
		}
	} else {
		p.flags = append(p.flags, p.current)
	}
	p.current = ""
	p.currentValues = nil
}

func (p *ruleParser) handleToken(token string) {
	switch {
	case strings.HasPrefix(token, "["):
		// 解析统计计数，如 [841:59388]
		p.packets, p.bytes, p.countersOk = parseCounters(token)
	case strings.HasPrefix(token, "-"):
		// 遇到新的选项，先保存之前的
		p.flush()
		p.current = token
	default:
		// 追加到当前参数列表
		p.currentValues = append(p.currentValues, token)
	}
}
