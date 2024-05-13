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

func TryBuild[T any](
	ctx context.Context,
	c *Container,
	name string,
	builderFc func(ctx context.Context, c *Container) (T, error),
) (T, error) {
	return TryBuildExtended(ctx, c, name, builderFc, nil, nil, false)
}

func Build[T any](
	ctx context.Context,
	c *Container,
	name string,
	builderFc func(ctx context.Context, c *Container) T,
) T {
	return BuildExtended(ctx, c, name, builderFc, nil, nil, false)
}

// SetService allow to bypass standard builder function and store mocked service into container.
// This is meant for testing purposes but can be also useful if you prefer to build some services outside of container.
func SetService(ctx context.Context, c *Container, name string, instance any, tags []string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	_, ok := c.storage[name]
	if ok {
		panic(fmt.Sprintf("service with name %s is already set", name))
	}
	c.storage[name] = &service{
		name:                  name,
		instance:              instance,
		isBuild:               true,
		hasCircularDependency: false,
		tags:                  tags,
	}
}

func TryBuildExtended[T any](
	ctx context.Context,
	c *Container,
	name string,
	builderFc func(ctx context.Context, c *Container) (T, error),
	tags []string,
	callbacks []func(ctx context.Context, c *Container, instance T) error,
	hasCircularDependency bool,
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
		if !s.isBuild && !s.hasCircularDependency {
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
		name:                  name,
		instance:              instance,
		isBuild:               false,
		hasCircularDependency: hasCircularDependency,
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

func BuildExtended[T any](
	ctx context.Context,
	c *Container,
	name string,
	builderFc func(ctx context.Context, c *Container) T,
	tags []string,
	callbacks []func(ctx context.Context, c *Container, instance T),
	hasCircularDependency bool,
) (instance T) {
	if ctx.Value(contextKey) == nil {
		c.lock.Lock()
		defer c.lock.Unlock()
		ctx = context.WithValue(ctx, contextKey, name)
	}

	if callbacks != nil && len(callbacks) > 0 {
		defer func() {
			for _, callback := range callbacks {
				callback(ctx, c, instance)
				return
			}
		}()
	}

	s, ok := c.storage[name]
	if ok {
		if !s.isBuild && !s.hasCircularDependency {
			panic(fmt.Sprintf("circular dependency detected with service %v", name))
		}
		var okType bool
		instance, okType = s.instance.(T)
		if !okType {
			panic("unable to assert type")
		}
		return instance
	}

	c.storage[name] = &service{
		name:                  name,
		instance:              instance,
		isBuild:               false,
		hasCircularDependency: hasCircularDependency,
		tags:                  tags,
	}

	instance = builderFc(ctx, c)

	c.storage[name].instance = instance
	c.storage[name].isBuild = true

	return instance
}
