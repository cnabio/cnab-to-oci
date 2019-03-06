package remotes

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestPromisesNominal(t *testing.T) {
	var mut sync.Mutex
	var n int
	var dependingValue int
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler := newErrgroupScheduler(ctx, 4, 5)
	var deps []dependency
	for i := 0; i < 4; i++ {
		deps = append(deps, scheduler.schedule(func(ctx context.Context) error {
			mut.Lock()
			defer mut.Unlock()
			n++
			return nil
		}))
	}
	final := newPromise(scheduler, whenAll(deps)).then(func(ctx context.Context) error {
		dependingValue = n
		return nil
	})
	assert.NilError(t, final.wait())
	assert.Equal(t, n, 4)
	assert.Equal(t, dependingValue, 4)
}

func TestPromisesCancelUnblock(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler := newErrgroupScheduler(ctx, 4, 5)
	var done bool
	started := make(chan struct{})
	p := scheduler.schedule(func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		done = true
		return nil
	})
	<-started
	cancel()
	assert.NilError(t, p.wait())
	assert.Check(t, done)
}

func TestPromisesErrorUnblock(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler := newErrgroupScheduler(ctx, 4, 5)
	var done bool
	p := scheduler.schedule(func(ctx context.Context) error {
		<-ctx.Done()
		done = true
		return nil
	})
	erroring := scheduler.schedule(func(ctx context.Context) error {
		return errors.New("boom")
	})
	assert.ErrorContains(t, erroring.wait(), "boom")
	assert.NilError(t, p.wait())
	assert.Check(t, done)
}

func TestPromisesScheduleErroredDontBlockDontRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler := newErrgroupScheduler(ctx, 4, 5)
	errErroringTask := scheduler.schedule(func(ctx context.Context) error {
		return errors.New("boom")
	}).wait()
	var done bool
	errAfterError := scheduler.schedule(func(ctx context.Context) error {
		done = true
		return nil
	}).wait()
	assert.ErrorContains(t, errErroringTask, "boom")
	assert.ErrorContains(t, errAfterError, "context canceled")
	assert.Check(t, !done)
}

func TestPromisesErrorUnblockDeps(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler := newErrgroupScheduler(ctx, 4, 5)
	dep := scheduler.schedule(func(ctx context.Context) error {
		time.Sleep(200)
		return errors.New("boom")
	})
	for i := 0; i < 50; i++ {
		dep = dep.then(func(ctx context.Context) error {
			return nil
		})
	}
	assert.ErrorContains(t, dep.wait(), "boom")
}

func TestPromisesUnwrwap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler := newErrgroupScheduler(ctx, 4, 5)
	var done1, done2 bool
	p := scheduleAndUnwrap(scheduler, func(ctx context.Context) (dependency, error) {
		done1 = true
		return scheduler.schedule(func(ctx context.Context) error {
			time.Sleep(200)
			done2 = true
			return nil
		}), nil
	})
	assert.NilError(t, p.wait())
	assert.Check(t, done1)
	assert.Check(t, done2)
}

func TestPromisesUnwrwapWithError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler := newErrgroupScheduler(ctx, 4, 5)
	var done bool
	p := scheduleAndUnwrap(scheduler, func(ctx context.Context) (dependency, error) {
		done = true
		return nil, errors.New("boom")
	})
	assert.ErrorContains(t, p.wait(), "boom")
	assert.Check(t, done)
}

func TestWhenAllWithErrorUnblocks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler := newErrgroupScheduler(ctx, 4, 5)
	var dependencies []dependency
	// add a blocking task
	dependencies = append(dependencies, scheduler.schedule(func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	}))
	dependencies = append(dependencies, failedDependency{errors.New("boom")})
	p := newPromise(scheduler, whenAll(dependencies)) // first error should be returned without waiting other
	// tasks to complete
	assert.ErrorContains(t, p.wait(), "boom")
}

func TestWhenAllWithErrorReported(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler := newErrgroupScheduler(ctx, 4, 5)
	var dependencies []dependency
	// add a blocking task
	dependencies = append(dependencies, doneDependency{})
	dependencies = append(dependencies, failedDependency{errors.New("boom")})
	p := newPromise(scheduler, whenAll(dependencies)) // first error should be returned without waiting other
	// tasks to complete
	assert.ErrorContains(t, p.wait(), "boom")
}
