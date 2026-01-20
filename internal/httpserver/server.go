package httpserver

import (
	"log/slog"
	"net"
	"net/http"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/sakthi1307/securelog/internal/middleware"
	"github.com/sakthi1307/securelog/internal/models"
	"github.com/sakthi1307/securelog/internal/store"
    "github.com/sakthi1307/securelog/internal/rules"

)


type Deps struct {
	Log    *slog.Logger
	APIKey string
	DB     *pgxpool.Pool
	RuleQueue chan<- rules.RuleEvalMsg

}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(middleware.AccessLog(d.Log))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/v1", func(v1 chi.Router) {

		v1.Use(middleware.APIKeyAuth(d.APIKey))
		v1.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ready"))
		})

		es := store.NewEventStore(d.DB)

		if d.DB == nil { panic("router deps: DB is nil") }
		if d.RuleQueue == nil { panic("router deps: RuleQueue is nil") }

		as := store.NewAlertStore(d.DB)

		publish := func(ev models.EventIngestDTO, ip net.IP) {
			if ip == nil || ev.Type == "" {
				return
			}
			// non-blocking send so ingest never deadlocks if queue fills
			select {
			case d.RuleQueue <- rules.RuleEvalMsg{EventType: ev.Type, SrcIP: ip, Ts: ev.Ts}:
			default:
			}
		}

		v1.Post("/events", IngestEventsHandler(es, publish))
		v1.Get("/alerts", ListAlertsHandler(as))
		v1.Post("/alerts/{id}/ack", AckAlertHandler(as))

	})

	return r
}
