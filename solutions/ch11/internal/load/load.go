// Package load drives a running banditd with simulated users and measures
// how it responds.
package load

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"banditlab/internal/bandit"
	"banditlab/internal/stat"
)

// Config describes one load test.
type Config struct {
	BaseURL string        // e.g. http://localhost:8080
	Workers int           // concurrent simulated users
	Elapsed time.Duration // how long to run; 0 means until ctx ends

	// Rate, if positive, switches to open-loop mode: selections arrive at
	// this many per second regardless of how fast the server answers.
	// Zero means closed-loop: each worker sends its next request as soon as
	// the previous one finishes.
	Rate float64

	Probs          []float64     // the world the users live in (per-arm success probability)
	Seed           uint64        // seeds each worker's random draws
	RewardFraction float64       // fraction of selections that get a reward call (0 is treated as 1)
	FeedbackDelay  time.Duration // pause between a selection and its reward

	Client *http.Client // nil means NewClient(Workers)
}

// NewClient returns an HTTP client suited to load generation: it keeps one
// idle connection per worker. The default transport keeps only two per host,
// so a busy load generator would keep opening and closing connections and
// mostly measure connection setup.
func NewClient(workers int) *http.Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.MaxIdleConns = workers
	tr.MaxIdleConnsPerHost = workers
	return &http.Client{Transport: tr, Timeout: 10 * time.Second}
}

// Stats summarises the calls to one endpoint.
type Stats struct {
	Count  int // completed calls, whatever the status
	Errors int // transport failures and non-2xx responses
	P50    time.Duration
	P90    time.Duration
	P99    time.Duration
	Max    time.Duration
}

// Report is the outcome of a run.
type Report struct {
	Elapsed    time.Duration
	Select     Stats
	Reward     Stats
	Status     map[int]int // responses by status code; 0 counts transport errors, -1 arms outside the scenario
	Throughput float64     // completed calls per second, both endpoints
	Dropped    int         // open-loop arrivals discarded because the backlog was full
	Pulls      []int       // how often the service chose each arm
	Regret     float64     // expected regret of those choices in Probs' world
}

// worker accumulates one goroutine's measurements; nothing here is shared,
// so no locking is needed until the results are merged.
type worker struct {
	env        *bandit.Env
	bestP      float64
	selLat     []time.Duration
	rewLat     []time.Duration
	selErrs    int
	rewErrs    int
	status     map[int]int
	pulls      []int
	regret     float64
	rewardRand func() float64
}

// Run executes the load test described by cfg.
func Run(ctx context.Context, cfg Config) (*Report, error) {
	if cfg.Workers < 1 || len(cfg.Probs) == 0 {
		return nil, fmt.Errorf("load: need at least one worker and one arm")
	}
	if cfg.RewardFraction == 0 {
		cfg.RewardFraction = 1
	}
	if cfg.Client == nil {
		cfg.Client = NewClient(cfg.Workers)
	}
	if cfg.Elapsed > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.Elapsed)
		defer cancel()
	}

	// Open loop: a dispatcher hands out arrival times; workers pick them up.
	var arrivals chan time.Time
	dropped := 0
	if cfg.Rate > 0 {
		arrivals = make(chan time.Time, 1<<14)
	}

	workers := make([]*worker, cfg.Workers)
	var wg sync.WaitGroup
	start := time.Now()
	for i := range workers {
		env, err := bandit.NewEnv(cfg.Probs, bandit.NewRNG(cfg.Seed, bandit.StreamEnv+uint64(1000+i)))
		if err != nil {
			return nil, err
		}
		_, bestP := env.Best()
		rewardRNG := bandit.NewRNG(cfg.Seed, uint64(5000+i))
		workers[i] = &worker{
			env:        env,
			bestP:      bestP,
			status:     map[int]int{},
			pulls:      make([]int, len(cfg.Probs)),
			rewardRand: rewardRNG.Float64,
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			workers[i].loop(ctx, cfg, arrivals)
		}()
	}
	if arrivals != nil {
		dropped = dispatch(ctx, cfg.Rate, start, arrivals)
		close(arrivals)
	}
	wg.Wait()
	return merge(workers, cfg, time.Since(start), dropped), nil
}

// dispatch sends arrival times at rate per second until ctx ends. If the
// workers fall behind and the channel fills, arrivals are dropped and counted.
func dispatch(ctx context.Context, rate float64, start time.Time, out chan<- time.Time) (dropped int) {
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	sent := 0
	for {
		select {
		case <-ctx.Done():
			return dropped
		case now := <-ticker.C:
			// Everything due by now, at its ideal time (not the time we noticed).
			due := int(now.Sub(start).Seconds() * rate)
			for ; sent < due; sent++ {
				at := start.Add(time.Duration(float64(sent+1) / rate * float64(time.Second)))
				select {
				case out <- at:
				default:
					dropped++
				}
			}
		}
	}
}

func (w *worker) loop(ctx context.Context, cfg Config, arrivals <-chan time.Time) {
	for ctx.Err() == nil {
		began := time.Now()
		if arrivals != nil {
			select {
			case scheduled, ok := <-arrivals:
				if !ok {
					return
				}
				// Latency counts from when the request *should* have started.
				// If the server is slow and requests queue up, the wait shows
				// up here instead of silently disappearing.
				began = scheduled
			case <-ctx.Done():
				return
			}
		}
		w.once(ctx, cfg, began)
	}
}

// once is one simulated user: get an arm, see whether it pays, report back.
func (w *worker) once(ctx context.Context, cfg Config, began time.Time) {
	id, arm, status, err := selectArm(ctx, cfg.Client, cfg.BaseURL)
	if ctx.Err() != nil {
		return // the run ended mid-request: do not count a cancelled call
	}
	w.selLat = append(w.selLat, time.Since(began))
	w.status[status]++
	if err != nil || status/100 != 2 {
		w.selErrs++
		return
	}
	if arm < 0 || arm >= len(w.pulls) {
		// The service has a different number of arms than the scenario.
		w.selErrs++
		w.status[-1]++ // -1: a well-formed response that makes no sense
		return
	}
	w.pulls[arm]++
	w.regret += w.bestP - w.env.Prob(arm)
	reward := w.env.Pull(arm)

	if cfg.RewardFraction < 1 && w.rewardRand() >= cfg.RewardFraction {
		return // this user never reports back
	}
	if cfg.FeedbackDelay > 0 {
		select {
		case <-time.After(cfg.FeedbackDelay):
		case <-ctx.Done():
			return
		}
	}
	t0 := time.Now()
	status, err = sendReward(ctx, cfg.Client, cfg.BaseURL, id, reward)
	if ctx.Err() != nil {
		return
	}
	w.rewLat = append(w.rewLat, time.Since(t0))
	w.status[status]++
	if err != nil || status/100 != 2 {
		w.rewErrs++
	}
}

func selectArm(ctx context.Context, c *http.Client, base string) (id string, arm, status int, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/select", nil)
	if err != nil {
		return "", 0, 0, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return "", 0, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body) // drain, so the connection can be reused
		return "", 0, resp.StatusCode, nil
	}
	var body struct {
		RequestID string `json:"request_id"`
		Arm       int    `json:"arm"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", 0, resp.StatusCode, err
	}
	return body.RequestID, body.Arm, resp.StatusCode, nil
}

func sendReward(ctx context.Context, c *http.Client, base, id string, reward float64) (status int, err error) {
	payload, _ := json.Marshal(map[string]any{"request_id": id, "reward": reward})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/reward", bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}

func merge(workers []*worker, cfg Config, elapsed time.Duration, dropped int) *Report {
	r := &Report{Elapsed: elapsed, Status: map[int]int{}, Dropped: dropped, Pulls: make([]int, len(cfg.Probs))}
	var sel, rew []float64
	for _, w := range workers {
		for _, d := range w.selLat {
			sel = append(sel, d.Seconds())
		}
		for _, d := range w.rewLat {
			rew = append(rew, d.Seconds())
		}
		r.Select.Errors += w.selErrs
		r.Reward.Errors += w.rewErrs
		for code, n := range w.status {
			r.Status[code] += n
		}
		for arm, n := range w.pulls {
			r.Pulls[arm] += n
		}
		r.Regret += w.regret
	}
	r.Select.fill(sel)
	r.Reward.fill(rew)
	if elapsed > 0 {
		r.Throughput = float64(r.Select.Count+r.Reward.Count) / elapsed.Seconds()
	}
	return r
}

// fill computes the summary from latencies in seconds.
func (s *Stats) fill(seconds []float64) {
	s.Count = len(seconds)
	if s.Count == 0 {
		return
	}
	at := func(q float64) time.Duration { return time.Duration(stat.Quantile(seconds, q) * float64(time.Second)) }
	s.P50, s.P90, s.P99, s.Max = at(0.5), at(0.9), at(0.99), at(1)
}

// Write prints the report as a small table.
func (r *Report) Write(w io.Writer) {
	fmt.Fprintf(w, "ran %v, %.0f requests/s in total\n\n", r.Elapsed.Round(time.Millisecond), r.Throughput)
	fmt.Fprintf(w, "%-8s %8s %7s %9s %9s %9s %9s\n", "call", "count", "errors", "p50", "p90", "p99", "max")
	for _, row := range []struct {
		name string
		s    Stats
	}{{"select", r.Select}, {"reward", r.Reward}} {
		fmt.Fprintf(w, "%-8s %8d %7d %9s %9s %9s %9s\n", row.name, row.s.Count, row.s.Errors,
			round(row.s.P50), round(row.s.P90), round(row.s.P99), round(row.s.Max))
	}
	if r.Dropped > 0 {
		fmt.Fprintf(w, "\n%d arrivals were dropped because the server could not keep up\n", r.Dropped)
	}
	fmt.Fprintf(w, "\narm choices: %v\n", r.Pulls)
	fmt.Fprintf(w, "expected regret of those choices: %.1f\n", r.Regret)
}

func round(d time.Duration) string {
	switch {
	case d >= time.Second:
		return d.Round(10 * time.Millisecond).String()
	case d >= time.Millisecond:
		return d.Round(10 * time.Microsecond).String()
	default:
		return d.Round(time.Microsecond).String()
	}
}
