package godic

import (
	"context"
	"testing"
)

func TestNewContainer(t *testing.T) {
	c := NewContainer()
	if c == nil {
		t.Fatalf("failed to create container")
	}
}

func TestCreateService(t *testing.T) {
	type A struct {
		name string
	}

	newA := func(name string) *A {
		return &A{
			name: name,
		}
	}

	type B struct {
		a *A
	}

	newB := func(a *A) *B {
		return &B{
			a: a,
		}
	}

	ctx := context.TODO()
	c := NewContainer()
	if c == nil {
		t.Fatalf("failed to create container")
	}

	a := func(ctx context.Context, c *Container) *A {
		return GetStrictBasic(ctx, c, "a", func(ctx context.Context, c *Container) (*A, error) {
			return newA("test"), nil
		},
		)
	}

	b := func(ctx context.Context, c *Container) *B {
		return GetStrictBasic(ctx, c, "b", func(ctx context.Context, c *Container) (*B, error) {
			return newB(a(ctx, c)), nil
		},
		)
	}
	bInstance := b(ctx, c)

	if b == nil || bInstance.a == nil {
		t.Fatalf("created service is nil")
	}

	if bInstance.a.name != "test" {
		t.Fatalf("service has wrong name provided %v, expected %v", bInstance.a.name, "test")
	}
}
