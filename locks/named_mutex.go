package locks

import "sync"

//go:generate counterfeiter -o ../fakes/named_locker.go --fake-name NamedLocker . NamedLocker
type NamedLocker interface {
	Lock(name string)
	Unlock(name string)
}

// TODO: this implementation leaks memory like a sieve
// perhaps we can use the filesystem instead of a map
type NamedMutex struct {
	control sync.Mutex

	namedLocks map[string]*sync.Mutex
}

func (g *NamedMutex) getOrCreateNamedLock(name string) *sync.Mutex {
	if g.namedLocks == nil {
		g.namedLocks = make(map[string]*sync.Mutex)
	}
	lock, ok := g.namedLocks[name]
	if !ok {
		lock = &sync.Mutex{}
		g.namedLocks[name] = lock
	}
	return lock
}

func (g *NamedMutex) Lock(name string) {
	g.control.Lock()
	namedLock := g.getOrCreateNamedLock(name)
	g.control.Unlock()
	namedLock.Lock()
}

func (g *NamedMutex) Unlock(name string) {
	g.control.Lock()
	namedLock := g.getOrCreateNamedLock(name)
	g.control.Unlock()
	namedLock.Unlock()
}