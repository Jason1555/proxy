package domain

import (
	"context"
)

type IPAccessRepository interface {
	GetPolicy(ctx context.Context) (*IPAccessPolicy, error)
	SavePolicy(ctx context.Context, policy *IPAccessPolicy) error
	AddEntry(ctx context.Context, entry *IPEntry) error
	RemoveEntry(ctx context.Context, id string) error
	GetEntry(ctx context.Context, id string) (*IPEntry, error)
	GetAllEntries(ctx context.Context, listType IPListType) ([]IPEntry, error)
	ClearList(ctx context.Context, listType IPListType) error
}
