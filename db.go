package main

import "sync"

type db struct {
	sync.Mutex
	m map[string]string
}

func (d *db) set(k, v string) bool {
	_, ok := d.get(k)
	if ok {
		return false
	}
	d.Lock()
	defer d.Unlock()
	d.m[k] = v
	return true
}

func (d *db) get(k string) (string, bool) {
	d.Lock()
	defer d.Unlock()
	v, ok := d.m[k]
	return v, ok
}

func (d *db) del(k string) bool {
	d.Lock()
	defer d.Unlock()
	delete(d.m, k)
	return true
}
