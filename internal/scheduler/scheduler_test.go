package scheduler

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeJob is a test Job whose behavior (panic, error, run count) is
// controllable. It signals each run on a channel so tests can synchronize
// without sleeping.
type fakeJob struct {
	name     string
	interval time.Duration
	runs     int32
	panicOn  int32 // panic when run count reaches this value (0 = never)
	errOn    int32 // return an error when run count reaches this value (0 = never)
	ran      chan struct{}
}

func newFakeJob(name string, interval time.Duration) *fakeJob {
	return &fakeJob{
		name:     name,
		interval: interval,
		ran:      make(chan struct{}, 100),
	}
}

func (j *fakeJob) Name() string            { return j.name }
func (j *fakeJob) Interval() time.Duration { return j.interval }

func (j *fakeJob) Run(ctx context.Context) error {
	n := atomic.AddInt32(&j.runs, 1)
	// Non-blocking signal so a stuck reader never deadlocks the job loop.
	select {
	case j.ran <- struct{}{}:
	default:
	}
	if p := atomic.LoadInt32(&j.panicOn); p != 0 && n == p {
		panic("fakeJob deliberate panic")
	}
	if e := atomic.LoadInt32(&j.errOn); e != 0 && n == e {
		return errors.New("fakeJob deliberate error")
	}
	return nil
}

// waitForRun blocks until the job reports a run or the deadline elapses.
func waitForRun(t *testing.T, j *fakeJob, timeout time.Duration) {
	t.Helper()
	select {
	case <-j.ran:
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for job %q to run", j.name)
	}
}

func TestSchedulerStartRunsImmediatelyThenStops(t *testing.T) {
	s := New(zerolog.Nop())
	// Long interval so the only run we observe is the immediate startup run.
	job := newFakeJob("immediate", time.Hour)
	s.Register(job)

	s.Start(context.Background())
	waitForRun(t, job, 2*time.Second)
	s.Stop() // must return cleanly (waits for the goroutine).

	assert.GreaterOrEqual(t, atomic.LoadInt32(&job.runs), int32(1),
		"job should run at least once immediately on Start")
}

func TestSchedulerRunsOnTicker(t *testing.T) {
	s := New(zerolog.Nop())
	job := newFakeJob("ticker", 20*time.Millisecond)
	s.Register(job)

	s.Start(context.Background())
	defer s.Stop()

	// Immediate run + at least one ticker-driven run.
	waitForRun(t, job, time.Second)
	waitForRun(t, job, time.Second)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&job.runs), int32(2))
}

func TestSchedulerRecoversFromPanickingJob(t *testing.T) {
	s := New(zerolog.Nop())
	// Panic on the first (immediate) run; the ticker loop must survive and
	// keep invoking the job afterwards.
	job := newFakeJob("panicker", 15*time.Millisecond)
	atomic.StoreInt32(&job.panicOn, 1)
	s.Register(job)

	s.Start(context.Background())
	defer s.Stop()

	// First run panics (and is recovered by executeJob).
	waitForRun(t, job, time.Second)
	// Subsequent ticker runs prove the loop did not crash.
	waitForRun(t, job, time.Second)
	waitForRun(t, job, time.Second)

	assert.GreaterOrEqual(t, atomic.LoadInt32(&job.runs), int32(3),
		"scheduler loop must keep running after a job panics")
}

func TestSchedulerErroringJobDoesNotCrashLoop(t *testing.T) {
	s := New(zerolog.Nop())
	job := newFakeJob("errorer", 15*time.Millisecond)
	atomic.StoreInt32(&job.errOn, 1)
	s.Register(job)

	s.Start(context.Background())
	defer s.Stop()

	waitForRun(t, job, time.Second)
	waitForRun(t, job, time.Second)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&job.runs), int32(2),
		"an erroring job should be logged but the loop must continue")
}

func TestSchedulerStopWithoutStartIsSafe(t *testing.T) {
	s := New(zerolog.Nop())
	// cancel is nil before Start; Stop must not panic.
	require.NotPanics(t, func() { s.Stop() })
}

func TestSchedulerStopCancelsViaParentContext(t *testing.T) {
	s := New(zerolog.Nop())
	job := newFakeJob("ctx", 10*time.Millisecond)
	s.Register(job)

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)
	waitForRun(t, job, time.Second)

	// Cancelling the parent context should stop the job loop; Stop then
	// returns cleanly without hanging.
	cancel()
	done := make(chan struct{})
	go func() { s.Stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not return after parent context cancellation")
	}
}
