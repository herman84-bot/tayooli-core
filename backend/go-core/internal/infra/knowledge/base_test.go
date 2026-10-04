package knowledge_test

import (
	"testing"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/knowledge"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/rag"
)

func TestGetAllDocuments_IncludesWMSAndPOS(t *testing.T) {
	docs := knowledge.GetAllDocuments()

	foundWMS := false
	foundPOS := false
	foundMarketplace := false

	for _, d := range docs {
		if d.ID == "produk-fitur-wms" {
			foundWMS = true
		}
		if d.ID == "produk-fitur-pos" {
			foundPOS = true
		}
		if d.ID == "produk-fitur-wms-marketplace" {
			foundMarketplace = true
		}
	}

	if !foundWMS {
		t.Errorf("expected produk-fitur-wms document in knowledge base")
	}
	if !foundPOS {
		t.Errorf("expected produk-fitur-pos document in knowledge base")
	}
	if !foundMarketplace {
		t.Errorf("expected produk-fitur-wms-marketplace document in knowledge base")
	}
}

func TestRAGSearch_FindsWMSAndPOS(t *testing.T) {
	kDocs := knowledge.GetAllDocuments()
	docs := make([]rag.Document, len(kDocs))
	for i, d := range kDocs {
		docs[i] = rag.Document{
			ID:      d.ID,
			Title:   d.Title,
			Content: d.Content,
			Tags:    d.Tags,
		}
	}

	engine := rag.NewSearchEngine(docs)

	// Test WMS query
	wmsResults := engine.Search("bagaimana cara transfer stok antar gudang di WMS?", 3)
	if len(wmsResults) == 0 {
		t.Fatalf("expected search results for WMS query, got none")
	}
	if wmsResults[0].Document.ID != "produk-fitur-wms" {
		t.Errorf("expected top result for WMS query to be produk-fitur-wms, got %s", wmsResults[0].Document.ID)
	}

	// Test POS query
	posResults := engine.Search("bagaimana cara transaksi kasir POS dan cetak struk thermal?", 3)
	if len(posResults) == 0 {
		t.Fatalf("expected search results for POS query, got none")
	}
	if posResults[0].Document.ID != "produk-fitur-pos" {
		t.Errorf("expected top result for POS query to be produk-fitur-pos, got %s", posResults[0].Document.ID)
	}
}
