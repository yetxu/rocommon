package util

import "strings"

const (
	INIT_TRIE_CHILDREN_NUM = 128 // Since we need to deal all kinds of language, so we use 128 instead of 26
)

// trieNode data structure
// trieNode itself doesn't have any value. The value is represented on the path
type trieNode struct {
	// if a node is the end of a word
	isEndOfWord bool

	// the collection of children of a node
	children map[rune]*trieNode
}

// Create new trieNode
func newtrieNode() *trieNode {
	return &trieNode{
		isEndOfWord: false,
		children:    make(map[rune]*trieNode, INIT_TRIE_CHILDREN_NUM),
	}
}

// Match index object
type matchIndex struct {
	start int // start index
	end   int // end index
}

// Construct from scratch
func newMatchIndex(start, end int) *matchIndex {
	return &matchIndex{
		start: start,
		end:   end,
	}
}

// Construct from existing match index object
func buildMatchIndex(obj *matchIndex) *matchIndex {
	return &matchIndex{
		start: obj.start,
		end:   obj.end,
	}
}

// dfa util
type DFAUtil struct {
	// The root node
	root *trieNode
}

func (a *DFAUtil) insertWord(word []rune) {
	currNode := a.root
	for _, c := range word {
		if cildNode, exist := currNode.children[c]; !exist {
			cildNode = newtrieNode()
			currNode.children[c] = cildNode
			currNode = cildNode
		} else {
			currNode = cildNode
		}
	}

	currNode.isEndOfWord = true
}

// Check if there is any word in the trie that starts with the given prefix.
func (a *DFAUtil) startsWith(prefix []rune) bool {
	currNode := a.root
	for _, c := range prefix {
		if cildNode, exist := currNode.children[c]; !exist {
			return false
		} else {
			currNode = cildNode
		}
	}

	return true
}

// Searc and make sure if a word is existed in the underlying trie.
func (a *DFAUtil) searcWord(word []rune) bool {
	currNode := a.root
	for _, c := range word {
		if cildNode, exist := currNode.children[c]; !exist {
			return false
		} else {
			currNode = cildNode
		}
	}

	return currNode.isEndOfWord
}

// Searc a whole sentence and get all the matcing words and their indices
// Return:
// A list of all the matc index object
func (a *DFAUtil) searcSentence(sentence string) (matchIndexList []*matchIndex) {
	start, end := 0, 1
	sentenceRuneList := []rune(sentence)

	// Iterate the sentence from the beginning to the end.
	startsWith := false
	for end <= len(sentenceRuneList) {
		// Check if a sensitive word starts with word range from [start:end)
		// We find the longest possible path
		// Then we check any sub word is the sensitive word from long to short
		if a.startsWith(sentenceRuneList[start:end]) {
			startsWith = true
			end += 1
		} else {
			if startsWith == true {
				// Check any sub word is the sensitive word from long to short
				for index := end - 1; index > start; index-- {
					if a.searcWord(sentenceRuneList[start:index]) {
						matchIndexList = append(matchIndexList, newMatchIndex(start, index-1))
						break
					}
				}
			}
			start, end = end-1, end+1
			startsWith = false
		}
	}

	// If finishing not because of unmatching, but reaching the end, we need to
	// check if the previous startsWith is true or not.
	// If it's true, we need to check if there is any candidate?
	if startsWith {
		for index := end - 1; index > start; index-- {
			if a.searcWord(sentenceRuneList[start:index]) {
				matchIndexList = append(matchIndexList, newMatchIndex(start, index-1))
				break
			}
		}
	}

	return
}

// Judge if input sentence contains some special caracter
// Return:
// Matc or not
func (a *DFAUtil) IsMatch(sentence string) bool {
	sentence = strings.TrimSpace(sentence)
	sentence = strings.Replace(sentence, " ", "", -1)
	sentence = strings.Replace(sentence, "&", "", -1)
	return len(a.searcSentence(sentence)) > 0
}

// Handle sentence. Use specified caracter to replace those sensitive caracters.
// input: Input sentence
// replaceCh: candidate
// Return:
// Sentence after manipulation
func (a *DFAUtil) HandleWord(sentence string, replaceCh rune) string {
	sentence1 := sentence
	sentence = strings.TrimSpace(sentence)
	sentence = strings.Replace(sentence, " ", "", -1)
	sentence = strings.Replace(sentence, "&", "", -1)
	matchIndexList := a.searcSentence(sentence)
	if len(matchIndexList) == 0 {
		return sentence1
	}

	// Manipulate
	sentenceList := []rune(sentence)
	for _, matchIndexObj := range matchIndexList {
		for index := matchIndexObj.start; index <= matchIndexObj.end; index++ {
			sentenceList[index] = replaceCh
		}
	}

	return string(sentenceList)
}

// Create new DfaUtil object
// wordList:word list
func NewDFAUtil(wordList []string) *DFAUtil {
	a := &DFAUtil{
		root: newtrieNode(),
	}

	for _, word := range wordList {
		wordRuneList := []rune(word)
		if len(wordRuneList) > 0 {
			a.insertWord(wordRuneList)
		}
	}

	return a
}

func DFAInsertWord(dfa *DFAUtil, wordList []string) {
	for _, word := range wordList {
		wordRuneList := []rune(word)
		if len(wordRuneList) > 0 {
			dfa.insertWord(wordRuneList)
		}
	}
}

//
//func TestIsMatch(t *testing.T) {
//	sensitiveList := []string{"中国", "中国人"}
//	input := "我来自中国cd"
//
//	util := NewDFAUtil(sensitiveList)
//	if util.IsMatch(input) == false {
//		t.Errorf("Expected true, but got false")
//	}
//}
//
//func TestHandleWord(t *testing.T) {
//	sensitiveList := []string{"中国", "中国人"}
//	input := "我来自中国cd"
//
//	util := NewDFAUtil(sensitiveList)
//	newInput := util.HandleWord(input, '*')
//	expected := "我来自**cd"
//	if newInput != expected {
//		t.Errorf("Expected %s, but got %s", expected, newInput)
//	}
//}
