package desensitizer

import "unicode/utf8"

// acOutput 记录在当前节点结束的模式及其字节深度。
type acOutput struct {
	pattern   string
	byteDepth int
}

// acNode 是 Aho-Corasick trie 的节点。
type acNode struct {
	children  map[rune]*acNode
	fail      *acNode
	outputs   []acOutput
	byteDepth int
}

// AhoCorasick 多模式字符串匹配器。
// 基于 trie 树 + 失败指针，时间复杂度 O(n + m + z)，其中：
//
//	n = 文本长度, m = 所有模式总长度, z = 匹配次数。
type AhoCorasick struct {
	root *acNode
}

// Match 表示一次命中。
type Match struct {
	Pattern string // 命中的完整关键词
	Start   int    // 在 text 中的起始字节位置（inclusive）
	End     int    // 结束字节位置（exclusive）
}

// NewAhoCorasick 从模式列表构建 AC 自动机。
func NewAhoCorasick(patterns []string) *AhoCorasick {
	ac := &AhoCorasick{
		root: &acNode{children: make(map[rune]*acNode)},
	}
	for _, p := range patterns {
		ac.insert(p)
	}
	ac.buildFail()
	return ac
}

func (ac *AhoCorasick) insert(pattern string) {
	node := ac.root
	for _, r := range pattern {
		if node.children[r] == nil {
			node.children[r] = &acNode{
				children:  make(map[rune]*acNode),
				byteDepth: node.byteDepth + utf8.RuneLen(r),
			}
		}
		node = node.children[r]
	}
	node.outputs = append(node.outputs, acOutput{
		pattern:   pattern,
		byteDepth: node.byteDepth,
	})
}

func (ac *AhoCorasick) buildFail() {
	queue := make([]*acNode, 0)
	for _, child := range ac.root.children {
		child.fail = ac.root
		queue = append(queue, child)
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for r, child := range current.children {
			failNode := current.fail
			for failNode != ac.root && failNode.children[r] == nil {
				failNode = failNode.fail
			}
			if next, ok := failNode.children[r]; ok && next != child {
				child.fail = next
			} else {
				child.fail = ac.root
			}
			child.outputs = append(child.outputs, child.fail.outputs...)
			queue = append(queue, child)
		}
	}
}

// Search 在 text 中搜索所有模式，返回命中列表。
func (ac *AhoCorasick) Search(text string) []Match {
	var matches []Match
	node := ac.root

	for i, r := range text {
		for node != ac.root && node.children[r] == nil {
			node = node.fail
		}
		if child, ok := node.children[r]; ok {
			node = child
		}

		for _, out := range node.outputs {
			end := i + utf8.RuneLen(r)
			start := end - out.byteDepth
			if start < 0 {
				start = 0
			}
			matches = append(matches, Match{
				Pattern: out.pattern,
				Start:   start,
				End:     end,
			})
		}
	}

	return matches
}
