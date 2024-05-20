package godic

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
)

func TestNewContainer(t *testing.T) {
	c := NewContainer()
	if c == nil {
		t.Fatalf("failed to create container")
	}
}

type A struct {
	id int
}

func NewA() *A {
	return &A{
		id: rand.Int(),
	}
}

type B struct {
	id int
	a  *A
}

func NewB(a *A) *B {
	return &B{
		id: rand.Int(),
		a:  a,
	}
}

type testCase struct {
	lock                                  sync.Mutex
	getAWasCalledTimes                    int
	getABuilderFuncWasCalledTimes         int
	getBWasCalledTimes                    int
	getBBuilderFuncWasCalledTimes         int
	getWithErrorWasCalledTimes            int
	getWithErrorBuilderFuncWasCalledTimes int
}

func (tc *testCase) GetA(ctx context.Context, c *Container) *A {
	tc.lock.Lock()
	tc.getAWasCalledTimes++
	tc.lock.Unlock()
	return Build(
		ctx,
		c,
		"a",
		func(ctx context.Context, c *Container) *A {
			tc.lock.Lock()
			tc.getABuilderFuncWasCalledTimes++
			tc.lock.Unlock()
			return NewA()
		},
	)
}

func (tc *testCase) GetB(ctx context.Context, c *Container) *B {
	tc.lock.Lock()
	tc.getBWasCalledTimes++
	tc.lock.Unlock()
	return Build(
		ctx,
		c,
		"b",
		func(ctx context.Context, c *Container) *B {
			tc.lock.Lock()
			tc.getBBuilderFuncWasCalledTimes++
			tc.lock.Unlock()
			return NewB(tc.GetA(ctx, c))
		},
	)
}

func (tc *testCase) GetBWithError(ctx context.Context, c *Container) (*B, error) {
	tc.lock.Lock()
	tc.getWithErrorWasCalledTimes++
	tc.lock.Unlock()
	return TryBuild(
		ctx,
		c,
		"b",
		func(ctx context.Context, c *Container) (*B, error) {
			tc.lock.Lock()
			tc.getWithErrorBuilderFuncWasCalledTimes++
			tc.lock.Unlock()
			return &B{}, fmt.Errorf("error")
		},
	)
}

func TestCreateService(t *testing.T) {
	tc := &testCase{}
	ctx := context.TODO()
	c := NewContainer()

	aInstance := tc.GetA(ctx, c)
	bInstance := tc.GetB(ctx, c)
	if bInstance == nil || bInstance.a == nil || aInstance == nil {
		t.Fatalf("created service is nil")
	}

	if bInstance.a.id != aInstance.id {
		t.Fatalf("service has wrong id provided %v, expected %v", bInstance.a.id, aInstance.id)
	}

	if tc.getAWasCalledTimes != 2 {
		t.Fatalf("getA method should be called 2 times")
	}

	if tc.getABuilderFuncWasCalledTimes != 1 {
		t.Fatalf("builder func for A should be called once")
	}

	if tc.getBWasCalledTimes != 1 {
		t.Fatalf("getB method should be called once")
	}

	if tc.getBBuilderFuncWasCalledTimes != 1 {
		t.Fatalf("builder func for B should be called once")
	}
}

func TestCreateServiceWithErr(t *testing.T) {
	tc := &testCase{}
	ctx := context.TODO()
	c := NewContainer()

	bInstance, err := tc.GetBWithError(ctx, c)
	if err == nil {
		t.Fatalf("error should not be nil")
	}
	bInstance, err = tc.GetBWithError(ctx, c)
	if err == nil {
		t.Fatalf("error should not be nil")
	}

	if bInstance != nil {
		t.Fatalf("bInstance should not be returned")
	}

	if tc.getWithErrorWasCalledTimes != tc.getWithErrorBuilderFuncWasCalledTimes {
		t.Fatalf("we did not reset container properly")
	}
}

func TestSetService(t *testing.T) {
	tc := &testCase{}
	ctx := context.TODO()
	c := NewContainer()
	SetService(c, "a", NewA())
	tc.GetA(ctx, c)

	if tc.getAWasCalledTimes != 1 && tc.getABuilderFuncWasCalledTimes != 0 {
		t.Fatalf("set service did not work properly")
	}
}

func TestConcurrency(t *testing.T) {
	tc := &testCase{}
	wg := &sync.WaitGroup{}
	ctx := context.TODO()
	c := NewContainer()

	p := func() {
		tc.GetA(ctx, c)
		wg.Done()
	}

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go p()
	}

	wg.Wait()

	if tc.getABuilderFuncWasCalledTimes != 1 || tc.getAWasCalledTimes != 1000 {
		t.Fatalf("container cache is not working properly")
	}
}
