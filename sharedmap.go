package edgex

import (
	"sync"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

type ExpiringMap struct {
	mm    map[string]*entry
	mutex *sync.RWMutex
}

func NewExpiringMap() *ExpiringMap {
	return &ExpiringMap{
		mm:    make(map[string]*entry),
		mutex: new(sync.RWMutex),
	}
}

// Add 设置Key、Value和缓存时间。返回是否添加成功。
// 如果Key已存在，添加将失败，并返回False。
func (slf *ExpiringMap) Add(key string, value interface{}, timeout time.Duration) bool {
	slf.mutex.Lock()
	defer slf.mutex.Unlock()

	_, found := slf.mm[key]
	if found {
		return false // Failed to add
	}
	var expire time.Time
	if timeout > 0 {
		expire = time.Now().Add(timeout)
	}
	slf.mm[key] = &entry{
		value:  value,
		expire: expire,
	}

	return true // Add success
}

// Get 获取Key的值。如果Key不存在，或者已超过缓存时间，返回 nil, false.
func (slf *ExpiringMap) Get(key string) (interface{}, bool) {
	slf.mutex.RLock()
	entry, exists := slf.mm[key]
	slf.mutex.RUnlock()

	if !exists {
		return nil, false
	} else {
		if !entry.expire.IsZero() && time.Now().After(entry.expire) {
			defer slf.Del(key)
			return nil, false
		} else {
			return entry.value, true
		}
	}
}

// Del 删除指定Key的值
func (slf *ExpiringMap) Del(key string) {
	slf.mutex.Lock()
	defer slf.mutex.Unlock()
	delete(slf.mm, key)
}

////

type entry struct {
	value  interface{}
	expire time.Time
}
