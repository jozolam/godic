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

func Strict[T any](ctx context.Context, c *Container, callback func(ctx context.Context, c *Container) (T, error)) T {
	instance, err := callback(ctx, c)
	if err != nil {
		panic(err)
	}

	return instance
}

func GetStrictBasic[T any](
	ctx context.Context,
	c *Container,
	name string,
	builderFc func(ctx context.Context, c *Container) (T, error),
) T {
	return GetStrict(ctx, c, name, builderFc, nil, nil, false)
}

func GetStrict[T any](
	ctx context.Context,
	c *Container,
	name string,
	builderFc func(ctx context.Context, c *Container) (T, error),
	tags []string,
	callbacks []func(ctx context.Context, c *Container, instance T) error,
	hasCircularDependency bool,
) T {
	instance, err := Get(ctx, c, name, builderFc, tags, callbacks, hasCircularDependency)
	if err != nil {
		panic(err)
	}

	return instance
}

func SetService(ctx context.Context, c *Container, name string, instance any, tags []string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	_, ok := c.storage[name]
	if ok {
		return fmt.Errorf("service with name %s is already set", name)
	}
	c.storage[name] = &service{
		name:                  name,
		instance:              instance,
		isBuild:               true,
		hasCircularDependency: false,
		tags:                  tags,
	}

	return nil
}

func Get[T any](
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
			return instance, fmt.Errorf("circular dependency detected with service %v", name)
		}
		var okType bool
		instance, okType = s.instance.(T)
		if !okType {
			return instance, fmt.Errorf("unable to assert type")
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
