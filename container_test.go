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

	ctx := context.TODO()
	c := NewContainer()
	if c == nil {
		t.Fatalf("failed to create container")
	}

	a, err := Get(
		ctx,
		c,
		"test",
		func(ctx context.Context, c *Container) (*A, error) {
			return newA("test"), nil
		},
		nil,
		nil,
	)

	if err != nil {
		t.Fatalf("failed to create service with error %v", err)
	}

	if a == nil {
		t.Fatalf("created service is nil")
	}

	if a.name != "test" {
		t.Fatalf("service has wrong name provided %v, expected %v", a.name, "test")
	}
}
