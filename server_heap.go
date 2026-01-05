package main

import (
	"container/heap"
	"sync"
)

// ServerHeap implements heap.Interface from the Go standard library
type ServerHeap []*Server

func (h ServerHeap) Len() int { return len(h) }

// Less is the key DSA logic: Sort by ActiveConnections (Min-Heap)
func (h ServerHeap) Less(i, j int) bool {
	return h[i].ActiveConnections < h[j].ActiveConnections
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
	old[n-1] = nil // Avoid memory leak
	item.Index = -1
	*h = old[0 : n-1]
	return item
}

// ==========================================
// Thread-Safe Pool Manager
// ==========================================

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
	// O(1) Operation - The best server is always at index 0
	return p.servers[0]
}

// IncrementActive updates the count and Re-Balances the Heap
func (p *ServerPool) IncrementActive(s *Server) {
	p.lock.Lock()
	defer p.lock.Unlock()

	s.ActiveConnections++
	// heap.Fix is O(log n) - It moves the server down the tree if it gets busy
	if s.Index != -1 {
		heap.Fix(&p.servers, s.Index)
	}
}

// DecrementActive updates the count and Re-Balances the Heap
func (p *ServerPool) DecrementActive(s *Server) {
	p.lock.Lock()
	defer p.lock.Unlock()

	s.ActiveConnections--
	// heap.Fix is O(log n) - It floats the server up the tree if it becomes free
	if s.Index != -1 {
		heap.Fix(&p.servers, s.Index)
	}
}
func (p *ServerPool) RemoveServer(s *Server) {
	p.lock.Lock()
	defer p.lock.Unlock()

	// Only remove if it's actually in the heap (Index != -1)
	if s.Index != -1 {
		heap.Remove(&p.servers, s.Index)
		s.Index = -1 // Mark as removed
	}
}
