package job

import (
	"fmt"
	"sync"
)

type signalMapNode struct {
	Chan     chan struct{}
	OnceDone sync.Once
	Done     chan struct{}
}

type signalMap struct {
	Data map[uint32]*signalMapNode
	Lock *sync.RWMutex
}

func newSignalMap(length int) *signalMap {
	jm := &signalMap{}
	jm.Data = make(map[uint32]*signalMapNode, length)
	jm.Lock = &sync.RWMutex{}
	return jm
}

func (jm *signalMap) Get(k uint32) (*signalMapNode, error) {
	jm.Lock.RLock()
	defer func() {
		jm.Lock.RUnlock()
	}()
	// jm[k]
	if _c, ok := jm.Data[k]; ok {
		return _c, nil
	}
	return nil, fmt.Errorf("KEY %v not found", k)
}

func (jm *signalMap) SendSignal(k uint32) error {
	jm.Lock.RLock()
	defer func() {
		jm.Lock.RUnlock()
	}()
	_j, ok := jm.Data[k]
	if !ok {
		return fmt.Errorf("KEY %v not found", k)
	}
	_j.Chan <- struct{}{}
	return nil
}

func (jm *signalMap) Create(k uint32) error {
	jm.Lock.Lock()
	defer func() {
		jm.Lock.Unlock()
	}()

	sigMN := &signalMapNode{}
	sigMN.Chan = make(chan struct{}, 1)
	sigMN.Done = make(chan struct{})
	jm.Data[k] = sigMN

	return nil
}

func (jm *signalMap) Close(k uint32) {
	jm.Lock.RLock()
	defer func() {
		jm.Lock.RUnlock()
	}()
	d, ok := jm.Data[k]
	if ok {
		d.OnceDone.Do(func() {
			close(d.Done)
			close(d.Chan)
		})
	}
}

func (jm *signalMap) Delete(k uint32) error {
	jm.Lock.Lock()
	defer func() {
		jm.Lock.Unlock()
	}()
	d, ok := jm.Data[k]
	if ok {
		// close(jm.Data[k])
		d.OnceDone.Do(func() {
			close(d.Done)
			close(d.Chan)
		})
	}
	delete(jm.Data, k)

	// jm.Data[k] = make(chan struct{}, 1)
	return nil
}
