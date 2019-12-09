package sysinfo

import "sync"

var hostname struct {
	sync.Mutex
	name string
}
