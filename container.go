package godic

import (
	"context"
	"fmt"
	"sync"
)

type Container struct {
	storage map[string]*service
	lock    sync.RWMutex
}

type service struct {
	name     string
	instance any
	isBuild  bool
}

func NewContainer() *Container {
	return &Container{
		storage: make(map[string]*service),
		lock:    sync.RWMutex{},
	}
}

const contextKey = "godic"

// SetService allow to bypass standard builder function and store mocked service into container.
// This is meant for testing purposes but can be also useful if you prefer to build some services outside of container.
func SetService(c *Container, name string, instance any) {
	c.lock.Lock()
	defer c.lock.Unlock()

	_, ok := c.storage[name]
	if ok {
		panic(fmt.Sprintf("service with name %s is already set", name))
	}
	c.storage[name] = &service{
		name:     name,
		instance: instance,
		isBuild:  true,
	}
}

func TryBuild[T any](
	ctx context.Context,
	c *Container,
	name string,
	builderFc func(ctx context.Context, c *Container) (T, error),
) (instance T, err error) {
	if ctx.Value(contextKey) == nil {
		c.lock.Lock()
		defer c.lock.Unlock()
		ctx = context.WithValue(ctx, contextKey, name)
	}

	s, ok := c.storage[name]
	if ok {
		if !s.isBuild {
			panic(fmt.Sprintf("circular dependency detected with service %v", name))
		}
		var okType bool

		instance, okType = s.instance.(T)
		if !okType {
			panic("unable to assert type")
		}
		return instance, nil
	}

	c.storage[name] = &service{
		name:     name,
		instance: instance,
		isBuild:  false,
	}

	temp, err := builderFc(ctx, c)
	if err != nil {
		delete(c.storage, name)
		return instance, err
	}
	instance = temp
	c.storage[name].instance = instance
	c.storage[name].isBuild = true

	return instance, nil
}

func Build[T any](
	ctx context.Context,
	c *Container,
	name string,
	builderFc func(ctx context.Context, c *Container) T,
) (instance T) {
	if ctx.Value(contextKey) == nil {
		c.lock.Lock()
		defer c.lock.Unlock()
		ctx = context.WithValue(ctx, contextKey, name)
	}

	s, ok := c.storage[name]
	if ok {
		if !s.isBuild {
			panic(fmt.Sprintf("circular dependency detected with services: %v and %v", name, s.name))
		}
		var okType bool
		instance, okType = s.instance.(T)
		if !okType {
			panic("unable to assert type")
		}
		return instance
	}

	c.storage[name] = &service{
		name:     name,
		instance: instance,
		isBuild:  false,
	}

	instance = builderFc(ctx, c)
	c.storage[name].instance = instance
	c.storage[name].isBuild = true

	return instance
}
