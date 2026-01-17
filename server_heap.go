package main

import (
	"container/heap"
	"sync"
)

type ServerHeap []*Server

func (h ServerHeap) Len() int { return len(h) }

// --- THE CORE DSA LOGIC ---
func (h ServerHeap) Less(i, j int) bool {
	// Weighted Least Connection Formula: ActiveConnections / Weight
	return float64(h[i].ActiveConnections)/float64(h[i].Weight) < float64(h[j].ActiveConnections)/float64(h[j].Weight)
}

func (h ServerHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].Index = i
	h[j].Index = j
}

func (h *ServerHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(*Server)
	item.Index = n
	*h = append(*h, item)
}

func (h *ServerHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	item.Index = -1
	*h = old[0 : n-1]
	return item
}

type ServerPool struct {
	servers ServerHeap
	lock    sync.Mutex
}

func (p *ServerPool) AddServer(s *Server) {
	p.lock.Lock()
	defer p.lock.Unlock()
	heap.Push(&p.servers, s)
}

func (p *ServerPool) GetNextServer() *Server {
	p.lock.Lock()
	defer p.lock.Unlock()
	if len(p.servers) == 0 {
		return nil
	}
	return p.servers[0]
}

func (p *ServerPool) IncrementActive(s *Server) {
	p.lock.Lock()
	defer p.lock.Unlock()
	s.ActiveConnections++
	if s.Index != -1 {
		heap.Fix(&p.servers, s.Index)
	}
}

func (p *ServerPool) DecrementActive(s *Server) {
	p.lock.Lock()
	defer p.lock.Unlock()
	s.ActiveConnections--
	if s.Index != -1 {
		heap.Fix(&p.servers, s.Index)
	}
}

func (p *ServerPool) RemoveServer(s *Server) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if s.Index != -1 {
		heap.Remove(&p.servers, s.Index)
		s.Index = -1
	}
}
