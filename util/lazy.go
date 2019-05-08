package util

import (
	"errors"
	"sync/atomic"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

type Lazy interface {
	Store(p interface{})
	Take(timeout time.Duration) (v interface{}, err error)
}

////

var ErrTakeTimeout = errors.New("take value timeout")

func NewLazyPacket() Lazy {
	return &lazy{
		state: 0,
	}
}

type lazy struct {
	Lazy
	value interface{}
	state uint32
}

func (l *lazy) Store(p interface{}) {
	l.value = p
	atomic.StoreUint32(&l.state, 1)
}

func (l *lazy) Take(timeout time.Duration) (v interface{}, err error) {
	done := make(chan struct{})
	go func() {
		for 0 == atomic.LoadUint32(&l.state) {
		}
		done <- struct{}{}
	}()

	select {
	case <-time.After(timeout):
		return nil, ErrTakeTimeout

	case <-done:
		atomic.StoreUint32(&l.state, 0)
		return l.value, nil
	}

}
