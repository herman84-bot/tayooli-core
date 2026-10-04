package rag

import (
	"context"
	"fmt"
	"strings"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/groq"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/knowledge"
)

const ragSystemPrompt = `Kamu adalah customer service AI Tayooli ERP. Kamu membantu user dengan pertanyaan tentang produk Tayooli ERP.

Karakter kamu:
- Ramah tapi profesional
- Bilingual: Bahasa Indonesia utama, Inggris jika user pakai Inggris
- Percakapan natural, tidak seperti robot
- Singkat dan to the point, tapi tetap sopan
- Gunakan "kamu" untuk user, "kami" untuk Tayooli
- Gunakan emoji seperlunya (maksimal 1-2 per pesan, jangan berlebihan)

Aturan penting:
- Kamu adalah AI yang membantu user 24/7
- Jawab BERDASARKAN konteks yang diberikan di bawah
- Jika konteks tidak cukup, jawab sebaik mungkin dari pengetahuan umum tentang ERP
- Jangan mengarang informasi spesifik yang tidak ada di konteks
- Gunakan informasi spesifik dari konteks (harga, langkah, fitur)
- Format jawaban dengan bullet point jika penjelasan panjang
- Selalu tawarkan bantuan lanjutan: "Ada yang lain yang bisa saya bantu?"`

// Pipeline implements RAG: search → inject context → generate.
type Pipeline struct {
	searchEngine *SearchEngine
	groqClient   *groq.Client
}

// NewPipeline creates a new RAG pipeline.
func NewPipeline(groqClient *groq.Client) *Pipeline {
	kDocs := knowledge.GetAllDocuments()
	docs := make([]Document, len(kDocs))
	for i, d := range kDocs {
		docs[i] = Document{
			ID:      d.ID,
			Title:   d.Title,
			Content: d.Content,
			Tags:    d.Tags,
		}
	}
	return &Pipeline{
		searchEngine: NewSearchEngine(docs),
		groqClient:   groqClient,
	}
}

// Query processes a user query through the RAG pipeline.
// Returns the AI response and the sources used.
func (p *Pipeline) Query(ctx context.Context, query string, history []groq.ChatMessage) (string, string, error) {
	// Step 1: Search for relevant documents
	results := p.searchEngine.Search(query, 5)

	// Step 2: Build context from search results
	context := p.buildContext(results)

	// Step 3: Build messages with augmented context
	messages := p.buildMessages(query, context, history)

	// Step 4: Generate response via Groq
	reply, model, err := p.groqClient.ChatCompletion(ctx, messages)
	if err != nil {
		return "", "", fmt.Errorf("groq error: %w", err)
	}

	// Step 5: Extract source titles
	sources := p.extractSources(results)

	return reply, model + " | sources: " + sources, nil
}

// buildContext creates a context string from search results.
func (p *Pipeline) buildContext(results []SearchResult) string {
	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Konteks dari dokumentasi Tayooli ERP:\n\n")

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("--- Dokumen %d: %s ---\n", i+1, r.Document.Title))
		sb.WriteString(r.Document.Content)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// buildMessages creates the message array for Groq with RAG context.
func (p *Pipeline) buildMessages(query, context string, history []groq.ChatMessage) []groq.ChatMessage {
	messages := []groq.ChatMessage{
		{Role: "system", Content: ragSystemPrompt},
	}

	// Add context if available
	if context != "" {
		messages = append(messages, groq.ChatMessage{
			Role:    "system",
			Content: context,
		})
	}

	// Add conversation history (last 10 messages)
	start := 0
	if len(history) > 10 {
		start = len(history) - 10
	}
	for _, m := range history[start:] {
		if m.Role != "system" {
			messages = append(messages, m)
		}
	}

	// Add current query
	messages = append(messages, groq.ChatMessage{
		Role:    "user",
		Content: query,
	})

	return messages
}

// extractSources returns a comma-separated list of source document titles.
func (p *Pipeline) extractSources(results []SearchResult) string {
	var titles []string
	for _, r := range results {
		titles = append(titles, r.Document.Title)
	}
	return strings.Join(titles, ", ")
}
