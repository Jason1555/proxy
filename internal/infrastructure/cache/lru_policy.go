package cache

import "container/list"

type lruPolicy struct {
	order *list.List
	cache map[string]*list.Element
}

func NewLRUPolicy() *lruPolicy {
	return &lruPolicy{
		order: list.New(),
		cache: make(map[string]*list.Element),
	}
}

func (p *lruPolicy) OnGet(key string) {
	if elem, exists := p.cache[key]; exists {
		p.order.MoveToFront(elem)
		return
	}
	p.cache[key] = p.order.PushFront(key)
}

func (p *lruPolicy) OnSet(key string, value any) {
	if elem, exists := p.cache[key]; exists {
		p.order.MoveToFront(elem)
		return
	}
	p.cache[key] = p.order.PushFront(key)
}

func (p *lruPolicy) OnDelete(key string) {
	if elem, exists := p.cache[key]; exists {
		p.order.Remove(elem)
		delete(p.cache, key)
	}
}

func (p *lruPolicy) GetVictim() string {
	if elem := p.order.Back(); elem != nil {
		return elem.Value.(string)
	}
	return ""
}
