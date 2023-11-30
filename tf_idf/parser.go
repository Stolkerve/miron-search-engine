package tfidf

import (
	"strings"
	"unicode"

	"github.com/Stolkerve.com/miron-search-engine/models"
)

const END_TOKEN = '\u0000'

type Parser struct {
	currentChar rune
	currentPos  uint
	readPos     uint
	input       []rune
}

func (p *Parser) readChar() {
	p.currentChar = p.peekChar()
	p.currentPos = p.readPos
	p.readPos += 1
}

func (p *Parser) peekChar() rune {
	if p.readPos >= uint(len(p.input)) {
		return END_TOKEN
	}
	return p.input[p.readPos]
}

func (p *Parser) eatWhitespace() {
	for {
		if unicode.IsSpace(p.currentChar) {
			p.readChar()
			continue
		}
		return
	}
}

func (p *Parser) parseWord() string {
	startPos := p.currentPos

	for unicode.IsLetter(p.currentChar) || unicode.IsNumber(p.currentChar) {
		p.readChar()
	}

	return string(p.input[startPos:p.currentPos])
}

func (p *Parser) parseNumber() string {
	startPos := p.currentPos

	for unicode.IsNumber(p.currentChar) && !unicode.IsSpace(p.currentChar) {
		p.readChar()
	}

	return string(p.input[startPos:p.currentPos])
}

func (p *Parser) ParseText(input string) (map[string]models.WordFreq, uint) {
	wordCount := uint(0)
	wordsFreq := make(map[string]models.WordFreq)

	p.input = []rune(input)
	p.readChar()

	for p.currentChar != END_TOKEN {
		p.eatWhitespace()

		var word string
		if unicode.IsLetter(p.currentChar) {
			word = p.parseWord()
		}
		if unicode.IsNumber(p.currentChar) {
			word = p.parseNumber()
		}

		word = strings.ToUpper(word)

		if len(word) != 0 {
			if wordFreq, ok := wordsFreq[word]; !ok {
				wordsFreq[word] = models.WordFreq{
					Word:  word,
					Count: 1,
				}
			} else {
				wordFreq.Count += 1
				wordsFreq[word] = wordFreq
			}
		}

		p.readChar()

		wordCount += 1
	}

	return wordsFreq, wordCount
}
