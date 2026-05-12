package repository

import (
	"context"
	"fmt"
	"proxy/internal/domain"
	"sync"
	"time"
)

type InMemoryIPAccessRepository struct {
	mu     sync.RWMutex
	policy *domain.IPAccessPolicy
}

func NewInMemoryIPAccessRepository() *InMemoryIPAccessRepository {
	return &InMemoryIPAccessRepository {
		policy: &domain.IPAccessPolicy {
			ID: "default",
			DefaultPolicy: "allow",
			AllowList: make([]domain.IPEntry, 0),
			DenyList: make([]domain.IPEntry, 0),
			GreyList: make([]domain.IPEntry, 0),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Version: 1,
		},
	}
}

func (r *InMemoryIPAccessRepository) GetPolicy(ctx context.Context) (*domain.IPAccessPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if r.policy == nil {
		return nil, fmt.Errorf("policy not found")
	}
	
	return r.policy, nil
}

func (r *InMemoryIPAccessRepository) SavePolicy(ctx context.Context, policy *domain.IPAccessPolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if policy == nil {
		return fmt.Errorf("policy cannot be nil")
	}
	
	policy.UpdatedAt = time.Now()
	policy.Version++
	
	r.policy = policy
	
	return nil
}

func (r *InMemoryIPAccessRepository) AddEntry(ctx context.Context, entry domain.IPEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	switch entry.Type {
	case domain.AllowList:
		r.policy.AllowList = append(r.policy.AllowList, entry)
	case domain.DenyList:
		r.policy.DenyList = append(r.policy.DenyList, entry)
	case domain.GreyList:
		r.policy.GreyList = append(r.policy.GreyList, entry)
	}

	r.policy.UpdatedAt = time.Now()
	r.policy.Version++
	
	return nil
}

func (r *InMemoryIPAccessRepository) RemoveEntry(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	found := false 
	
	for i, entry := range r.policy.AllowList {
		if entry.ID == id {
			r.policy.AllowList = append(r.policy.AllowList[:i], r.policy.AllowList[i+1:]...)
			found = true
			break
		}
	}
	
	if !found {
		for i, entry := range r.policy.GreyList {
			if entry.ID == id {
				r.policy.GreyList = append(r.policy.GreyList[:i], r.policy.GreyList[i+1:]...)
				found = true
				break
			}
		}
	}
	
	if !found {
		return fmt.Errorf("entry not found: %s", id)
	}

	r.policy.UpdatedAt = time.Now()
	r.policy.Version++
	
	return nil
}

func (r *InMemoryIPAccessRepository) GetEntry(ctx context.Context, id string) (*domain.IPEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	for i := range r.policy.AllowList {
		if r.policy.AllowList[i].ID == id {
			return &r.policy.AllowList[i], nil
		}
	}
	
	for i := range r.policy.DenyList {
		if r.policy.DenyList[i].ID == id {
			return &r.policy.DenyList[i], nil
		}
	}
	
	for i := range r.policy.GreyList {
		if r.policy.GreyList[i].ID == id {
			return &r.policy.GreyList[i], nil
		}
	}
	
	return nil, fmt.Errorf("entry not found: %s", id)
}

func (r *InMemoryIPAccessRepository) GetAllEntries(ctx context.Context, listType domain.IPListType) ([]domain.IPEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	switch listType{
	case domain.AllowList:
		return r.policy.AllowList, nil
	case domain.DenyList:
		return r.policy.DenyList, nil
	case domain.GreyList:
		return r.policy.GreyList, nil
	default:
		return nil, fmt.Errorf("invalid list type: %s", listType)
	}
}

func (r *InMemoryIPAccessRepository) ClearList(ctx context.Context, listType domain.IPListType) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	switch listType {
	case domain.AllowList:
		r.policy.AllowList = make([]domain.IPEntry, 0)
	case domain.DenyList:
		r.policy.DenyList = make([]domain.IPEntry, 0)
	case domain.GreyList:
		r.policy.GreyList = make([]domain.IPEntry, 0)
	default:
		return fmt.Errorf("invalid list type: %s", listType)
	}

	r.policy.UpdatedAt = time.Now()
	r.policy.Version++
	
	return nil
}
