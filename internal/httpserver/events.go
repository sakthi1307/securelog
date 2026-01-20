package httpserver

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/sakthi1307/securelog/internal/models"
	"github.com/sakthi1307/securelog/internal/store"
)

type ingestRequest struct {
	Events []models.EventIngestDTO `json:"events"`
}

type ingestItemResult struct {
	Index  int    `json:"index"`
	Status string `json:"status"`          // "ok" or "error"
	ID     string `json:"id,omitempty"`    // inserted event id
	Error  string `json:"error,omitempty"` // error message if any
}

type EventJob struct {
	Index int
	Event models.EventIngestDTO
}

func IngestEventsHandler(es *store.EventStore, publish func(models.EventIngestDTO, net.IP)) http.HandlerFunc{
	// Worker pool sizing: small & safe default
	workers := max(2, runtime.GOMAXPROCS(0))

	return func(w http.ResponseWriter, r *http.Request) {
		// Safety: avoid huge payloads
		r.Body = http.MaxBytesReader(w, r.Body, 2<<20) // 2MB

		var req ingestRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if len(req.Events) == 0 {
			http.Error(w, "events is required", http.StatusBadRequest)
			return
		}
		if len(req.Events) > 5000 {
			http.Error(w, "too many events (max 5000)", http.StatusBadRequest)
			return
		}

		jobs := make(chan EventJob, len(req.Events))
		results := make(chan ingestItemResult, len(req.Events))

		ctx := r.Context()

		var wg sync.WaitGroup
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func() {
				defer wg.Done()
				worker(ctx, es, jobs, results,publish)
			}()
		}

		// enqueue jobs
		for i, ev := range req.Events {
			jobs <- EventJob{Index: i, Event: ev}
		}
		close(jobs)

		// close results after workers done
		go func() {
			wg.Wait()
			close(results)
		}()

		out := make([]ingestItemResult, 0, len(req.Events))
		for res := range results {
			out = append(out, res)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": out,
		})
	}
}

func worker(
    ctx context.Context,
    es *store.EventStore,
    jobs <-chan EventJob,
    results chan<- ingestItemResult,
    publish func(models.EventIngestDTO, net.IP),
) {
	for job := range jobs {
		ev := job.Event
		ip, ok, msg := ev.Validate()
		if !ok {
			results <- ingestItemResult{Index: job.Index, Status: "error", Error: msg}
			continue
		}

		dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		ins, err := es.Insert(dbCtx, ev, ip)
		cancel()
		
		if err != nil {
			results <- ingestItemResult{Index: job.Index, Status: "error", Error: "db insert failed"}
			continue
		}
		if publish != nil {
			publish(ev, ip)
		}
		results <- ingestItemResult{Index: job.Index, Status: "ok", ID: ins.ID}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
