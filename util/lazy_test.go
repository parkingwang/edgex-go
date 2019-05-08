package util

import (
	"fmt"
	"testing"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

func TestLazy_Take(t *testing.T) {
	lz := NewLazyPacket()
	time.AfterFunc(time.Millisecond*3, func() {
		lz.Store("HAHA")
	})
	if _, err := lz.Take(time.Millisecond); nil == err {
		t.Error("Should timeout 1")
	}
	if _, err := lz.Take(time.Millisecond); nil == err {
		t.Error("Should timeout 2")
	}
	if v, err := lz.Take(time.Millisecond * 2); nil != err {
		t.Error("Should take, error: ", err)
	} else {
		fmt.Println("Take: ", v)
	}
}
