package ex2

import (
	"context"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var info = &grpc.UnaryServerInfo{FullMethod: "/test/Method"}

func TestRejectsWhenFull(t *testing.T) {
	ic := MaxInFlight(2)
	if ic == nil {
		t.Fatal("MaxInFlight returned nil")
	}
	release := make(chan struct{})
	entered := make(chan struct{}, 2)
	slow := func(context.Context, any) (any, error) {
		entered <- struct{}{}
		<-release
		return "done", nil
	}

	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if resp, err := ic(context.Background(), nil, info, slow); err != nil || resp != "done" {
				t.Errorf("admitted call = %v, %v", resp, err)
			}
		}()
	}
	<-entered
	<-entered // both slots are taken

	called := false
	_, err := ic(context.Background(), nil, info, func(context.Context, any) (any, error) {
		called = true
		return nil, nil
	})
	if status.Code(err) != codes.ResourceExhausted || called {
		t.Errorf("third call: err = %v, handler called = %v; want ResourceExhausted and no call", err, called)
	}

	close(release)
	wg.Wait()
	if _, err := ic(context.Background(), nil, info, func(context.Context, any) (any, error) { return nil, nil }); err != nil {
		t.Errorf("after the others finished: %v, want success", err)
	}
}

func TestSlotIsReturnedAfterErrorsAndPanics(t *testing.T) {
	ic := MaxInFlight(1)
	fail := func(context.Context, any) (any, error) { return nil, status.Error(codes.Internal, "x") }
	boom := func(context.Context, any) (any, error) { panic("boom") }
	ok := func(context.Context, any) (any, error) { return 1, nil }

	for range 3 {
		ic(context.Background(), nil, info, fail)
	}
	func() {
		defer func() { recover() }()
		ic(context.Background(), nil, info, boom)
	}()
	if _, err := ic(context.Background(), nil, info, ok); err != nil {
		t.Errorf("slot leaked: %v", err)
	}
}

func TestNeverExceedsLimitUnderLoad(t *testing.T) {
	const limit = 4
	ic := MaxInFlight(limit)
	var mu sync.Mutex
	running, peak := 0, 0
	h := func(context.Context, any) (any, error) {
		mu.Lock()
		running++
		peak = max(peak, running)
		mu.Unlock()
		for range 100 {
			mu.Lock()
			mu.Unlock()
		}
		mu.Lock()
		running--
		mu.Unlock()
		return nil, nil
	}
	var wg sync.WaitGroup
	for range 64 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ic(context.Background(), nil, info, h)
		}()
	}
	wg.Wait()
	if peak > limit {
		t.Errorf("%d calls ran at once, limit is %d", peak, limit)
	}
}
