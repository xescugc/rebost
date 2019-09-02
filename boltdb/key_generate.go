package boltdb

import (
	"fmt"
	"sync"
	"time"
)

// keyGenerator holds the key generator
type keyGenerator struct {
	sync.Mutex

	// lastKey is the last key generated, so if thy happen at the
	// same time it can increase it without duplication
	lastKey int64
}

var newKey func() []byte

func init() {
	k := keyGenerator{}
	newKey = k.new
}

// new genrates a new unique incrementing key
func (k *keyGenerator) new() []byte {
	k.Lock()
	defer k.Unlock()
	t := time.Now().UnixNano()
	if t <= k.lastKey {
		t = k.lastKey + 1
	}
	k.lastKey = t
	return []byte(fmt.Sprintf("%d", t))
}
