package rag

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

// Document represents a searchable document.
type Document struct {
	ID      string
	Title   string
	Content string
	Tags    []string
}

// SearchResult represents a search result with relevance score.
type SearchResult struct {
	Document Document
	Score    float64
}

// SearchEngine implements BM25-style keyword search.
type SearchEngine struct {
	documents []Document
	k1        float64
	b         float64
	avgDl     float64
	docLens   []int
	idf       map[string]float64
	index     map[string][]int // term -> list of doc indices
}

// NewSearchEngine creates a new search engine with the given documents.
func NewSearchEngine(docs []Document) *SearchEngine {
	se := &SearchEngine{
		documents: docs,
		k1:        1.5,
		b:         0.75,
	}
	se.buildIndex()
	return se
}

// Search finds the top-K most relevant documents for a query.
func (se *SearchEngine) Search(query string, topK int) []SearchResult {
	queryTerms := tokenize(query)
	if len(queryTerms) == 0 {
		return nil
	}

	scores := make([]float64, len(se.documents))

	for _, term := range queryTerms {
		if _, ok := se.idf[term]; !ok {
			continue
		}

		idf := se.idf[term]
		docIndices, ok := se.index[term]
		if !ok {
			continue
		}

		for _, docIdx := range docIndices {
			dl := se.docLens[docIdx]
			tf := float64(strings.Count(se.documents[docIdx].Content, term))
			// Also search in title and tags
			tf += float64(strings.Count(se.documents[docIdx].Title, term)) * 2.0
			for _, tag := range se.documents[docIdx].Tags {
				if strings.Contains(tag, term) {
					tf += 3.0
				}
			}

			score := idf * (tf * (se.k1 + 1)) / (tf + se.k1*(1-se.b+se.b*float64(dl)/se.avgDl))
			scores[docIdx] += score
		}
	}

	// Collect results with non-zero scores
	results := make([]SearchResult, 0)
	for i, score := range scores {
		if score > 0 {
			results = append(results, SearchResult{
				Document: se.documents[i],
				Score:    score,
			})
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Return top-K
	if len(results) > topK {
		results = results[:topK]
	}

	return results
}

// buildIndex builds the inverted index and computes IDF.
func (se *SearchEngine) buildIndex() {
	n := float64(len(se.documents))
	se.index = make(map[string][]int)
	se.idf = make(map[string]float64)
	se.docLens = make([]int, len(se.documents))

	totalLen := 0
	df := make(map[string]int) // document frequency

	for i, doc := range se.documents {
		text := doc.Title + " " + doc.Content + " " + strings.Join(doc.Tags, " ")
		terms := tokenize(text)
		se.docLens[i] = len(terms)
		totalLen += len(terms)

		// Unique terms in this document
		seen := make(map[string]bool)
		for _, term := range terms {
			if !seen[term] {
				seen[term] = true
				df[term]++
				se.index[term] = append(se.index[term], i)
			}
		}
	}

	if n > 0 {
		se.avgDl = float64(totalLen) / n
	}

	// Compute IDF: log((N - df + 0.5) / (df + 0.5) + 1)
	for term, docFreq := range df {
		se.idf[term] = math.Log((n-float64(docFreq)+0.5)/(float64(docFreq)+0.5) + 1)
	}
}

// tokenize splits text into lowercase terms.
func tokenize(text string) []string {
	text = strings.ToLower(text)
	var terms []string
	var current strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				terms = append(terms, current.String())
				current.Reset()
			}
		}
	}
	if current.Len() > 0 {
		terms = append(terms, current.String())
	}

	return terms
}
