package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKnowledgeTokenizer_Tokenize(t *testing.T) {
	tok := NewKnowledgeTokenizer()
	freq := tok.Tokenize("感冒和发热症状")
	assert.Greater(t, freq["感冒"], 0)
	assert.Greater(t, freq["发热"], 0)
	assert.Greater(t, freq["症状"], 0)
}

func TestKnowledgeTokenizer_TokenizeASCII(t *testing.T) {
	tok := NewKnowledgeTokenizer()
	freq := tok.Tokenize("cold and fever symptoms")
	assert.Greater(t, freq["cold"], 0)
	assert.Greater(t, freq["fever"], 0)
	assert.Greater(t, freq["symptoms"], 0)
	assert.Equal(t, 0, freq["and"]) // stopword
}
