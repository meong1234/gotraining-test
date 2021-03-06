package pool

import (
	"errors"
	"io"
	"log"
	"sync"
)

type Pool struct {
	mu        sync.Mutex
	resources chan io.Closer
	factory   func() (io.Closer, error)
	closed    bool
}

var ErrPoolClosed = errors.New("pool has been closed")

func New(size uint, f func() (io.Closer, error)) (*Pool, error) {

	if size == 0 {
		return nil, errors.New("size value to small")
	}

	return &Pool{
		factory:   f,
		resources: make(chan io.Closer, size),
	}, nil
}

func (p *Pool) Acquire() (io.Closer, error) {

	select {
	case r, wd := <-p.resources:
		log.Println("acquire", "shared resource")
		if !wd {
			return nil, ErrPoolClosed
		}

		return r, nil
	default:
		log.Println("acquire", "new resource")
		return p.factory()
	}
}

func (p *Pool) Release(r io.Closer) {

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		r.Close()
		return
	}

	select {
	case p.resources <- r:
		log.Println("Release", "in queue")
	default:
		log.Println("Release", "Closing")
		r.Close()
	}
}

func (p *Pool) Close() error {

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrPoolClosed
	}

	p.closed = true

	close(p.resources)

	for r := range p.resources {
		r.Close()
	}
	return nil
}
