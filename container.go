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
	name                  string
	instance              any
	isBuild               bool
	hasCircularDependency bool
	tags                  []string
}

func NewContainer() *Container {
	return &Container{
		storage: make(map[string]*service),
		lock:    sync.RWMutex{},
	}
}

const contextKey = "something"

func GetStrict[T any](
	ctx context.Context,
	c *Container,
	name string,
	builderFc func(ctx context.Context, c *Container) (T, error),
	tags []string,
	callbacks []func(ctx context.Context, c *Container, instance T) error,
) T {
	instance, err := Get(ctx, c, name, builderFc, tags, callbacks)
	if err != nil {
		panic(err)
	}

	return instance
}

func Get[T any](
	ctx context.Context,
	c *Container,
	name string,
	builderFc func(ctx context.Context, c *Container) (T, error),
	tags []string,
	callbacks []func(ctx context.Context, c *Container, instance T) error,
) (instance T, err error) {
	if ctx.Value(contextKey) == nil {
		c.lock.Lock()
		defer c.lock.Unlock()
		ctx = context.WithValue(ctx, contextKey, name)
	}

	if callbacks != nil && len(callbacks) > 0 {
		defer func() {
			if err == nil {
				for _, callback := range callbacks {
					err = callback(ctx, c, instance)
					return
				}
			}
		}()
	}

	s, ok := c.storage[name]
	if ok {
		if !s.isBuild {
			return instance, fmt.Errorf("circular dependency detected with service %v", name)
		}
		instance, okType := s.instance.(T)
		if !okType {
			return instance, fmt.Errorf("unable to assert type")
		}
		return instance, nil
	}

	c.storage[name] = &service{
		name:                  name,
		instance:              instance,
		isBuild:               false,
		hasCircularDependency: false,
		tags:                  tags,
	}

	instance, err = builderFc(ctx, c)
	if err != nil {
		delete(c.storage, name)
		return instance, err
	}

	c.storage[name].instance = instance
	c.storage[name].isBuild = true

	return instance, nil
}
