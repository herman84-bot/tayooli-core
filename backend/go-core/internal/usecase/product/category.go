package product

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// normalizeCategoryName trims and collapses inner whitespace, then enforces
// 1..ProductCategoryMaxNameLen characters (the column is VARCHAR(100)).
func normalizeCategoryName(raw string) (string, error) {
	name := strings.Join(strings.Fields(raw), " ")
	if name == "" || utf8.RuneCountInString(name) > domain.ProductCategoryMaxNameLen {
		return "", domain.ErrInvalidInput
	}
	return name, nil
}

func (u *Usecase) requireCategories() error {
	if u.categoryRepo == nil {
		return domain.ErrInvalidInput
	}
	return nil
}

func (u *Usecase) ListCategories(ctx context.Context, tenantID uuid.UUID) ([]domain.ProductCategory, error) {
	if err := u.requireCategories(); err != nil {
		return nil, err
	}
	return u.categoryRepo.List(ctx, tenantID)
}

// CreateCategory returns ErrConflict when the name already exists in the
// tenant (case-insensitive).
func (u *Usecase) CreateCategory(ctx context.Context, tenantID uuid.UUID, rawName string) (*domain.ProductCategory, error) {
	if err := u.requireCategories(); err != nil {
		return nil, err
	}
	name, err := normalizeCategoryName(rawName)
	if err != nil {
		return nil, err
	}
	c := &domain.ProductCategory{ID: uuid.New(), TenantID: tenantID, Name: name, CreatedAt: time.Now().UTC()}
	if err := u.categoryRepo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (u *Usecase) RenameCategory(ctx context.Context, tenantID, id uuid.UUID, rawName string) (*domain.ProductCategory, error) {
	if err := u.requireCategories(); err != nil {
		return nil, err
	}
	name, err := normalizeCategoryName(rawName)
	if err != nil {
		return nil, err
	}
	if err := u.categoryRepo.Rename(ctx, tenantID, id, name); err != nil {
		return nil, err
	}
	return u.categoryRepo.GetByID(ctx, tenantID, id)
}

// DeleteCategory removes a category; its products become uncategorized.
func (u *Usecase) DeleteCategory(ctx context.Context, tenantID, id uuid.UUID) error {
	if err := u.requireCategories(); err != nil {
		return err
	}
	return u.categoryRepo.Delete(ctx, tenantID, id)
}
