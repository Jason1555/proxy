package repository

import (
	"context"
	"proxy/internal/domain"
)

type IPAccessRepository interface {
	GetPolicy(ctx context.Context) (*domain.IPAccessPolicy, error)
	SavePolicy(ctx context.Context, policy *domain.IPAccessPolicy) error
	AddEntry(ctx context.Context, entry *domain.IPEntry) error
	RemoveEntry(ctx context.Context, id string) error
	GetEntry(ctx context.Context, id string) (*domain.IPEntry, error)
	GetAllEntries(ctx context.Context, listType domain.IPListType) ([]domain.IPEntry, error)
	ClearList(ctx context.Context, listType domain.IPListType) error
}
