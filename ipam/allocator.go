package ipam

import (
	"errors"
	"fmt"
	"math/big"
	"net"
	"sync"

	"github.com/appc/cni/pkg/types"
)

//go:generate counterfeiter -o ../fakes/allocator_store.go --fake-name AllocatorStore . AllocatorStore
type AllocatorStore interface {
	Reserve(id string, ip net.IP) (bool, error)
	ReleaseByID(id string) error
}

var NoMoreAddressesError = errors.New("no addresses available")

//go:generate counterfeiter -o ../fakes/store_factory.go --fake-name StoreFactory . storeFactory
type storeFactory interface {
	Create(path string) (AllocatorStore, error)
}

//go:generate counterfeiter -o ../fakes/config_factory.go --fake-name ConfigFactory . configFactory
type configFactory interface {
	Create(networkID string) (types.IPConfig, error)
}

//go:generate counterfeiter -o ../fakes/locker.go --fake-name Locker . locker
type locker interface {
	sync.Locker
}

type allocator struct {
	storeFactory storeFactory
	storeLocker  locker
	stores       map[string]AllocatorStore

	configFactory configFactory
	configLocker  locker
	configs       map[string]*types.IPConfig
}

//go:generate counterfeiter -o ../fakes/ip_allocator.go --fake-name IPAllocator . ipAllocator
type IPAllocator interface {
	AllocateIP(networkID, containerID string) (*types.Result, error)
	ReleaseIP(networkID, containerID string) error
}

func New(storeFactory storeFactory, storeLocker locker, configFactory configFactory, configLocker sync.Locker) IPAllocator {
	return &allocator{
		storeFactory: storeFactory,
		storeLocker:  storeLocker,
		stores:       map[string]AllocatorStore{},

		configFactory: configFactory,
		configLocker:  configLocker,
		configs:       map[string]*types.IPConfig{},
	}
}

func (a *allocator) AllocateIP(networkID, containerID string) (*types.Result, error) {
	config, err := a.getConfig(networkID)
	if err != nil {
		return nil, err
	}

	store, err := a.getStore(networkID)
	if err != nil {
		return nil, err
	}

	ip := config.IP.IP

	if config.Gateway == nil {
		ip = nextIP(ip)
		config.Gateway = ip
	}

	for {
		ip = nextIP(ip)

		if !config.IP.Contains(ip) {
			return nil, NoMoreAddressesError
		}

		if config.Gateway.Equal(ip) {
			continue
		}

		ok, err := store.Reserve(containerID, ip)
		if err != nil {
			return nil, fmt.Errorf("failed to reserve IP: %s", err)
		}

		if ok {
			break
		}
	}

	result := &types.IPConfig{
		IP: net.IPNet{
			IP:   ip,
			Mask: config.IP.Mask,
		},
		Gateway: config.Gateway,
		Routes:  config.Routes,
	}

	return &types.Result{IP4: result}, nil
}

func (a *allocator) ReleaseIP(networkID, containerID string) error {
	store, err := a.getStore(networkID)
	if err != nil {
		return err
	}

	err = store.ReleaseByID(containerID)
	if err != nil {
		return fmt.Errorf("store failed to release: %s", err)
	}

	return nil
}

func (a *allocator) getConfig(networkID string) (*types.IPConfig, error) {
	a.configLocker.Lock()
	defer a.configLocker.Unlock()

	if config := a.configs[networkID]; config != nil {
		return config, nil
	}

	config, err := a.configFactory.Create(networkID)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain configuration for network %q: %s", networkID, err)
	}
	a.configs[networkID] = &config

	return &config, err
}

func (a *allocator) getStore(networkID string) (AllocatorStore, error) {
	a.storeLocker.Lock()
	defer a.storeLocker.Unlock()

	if store := a.stores[networkID]; store != nil {
		return store, nil
	}

	store, err := a.storeFactory.Create(networkID)
	if err != nil {
		return nil, fmt.Errorf("failed to create allocator store: %s", err)
	}
	a.stores[networkID] = store

	return store, nil
}

func nextIP(ip net.IP) net.IP {
	newIPInt := big.NewInt(0).SetBytes(ip.To4())
	newIPInt.Add(newIPInt, big.NewInt(1))
	return net.IP(newIPInt.Bytes())
}
