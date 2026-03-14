package owner

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SSEHandler struct {
	db *pgxpool.Pool
}

func NewSSEHandler(db *pgxpool.Pool) *SSEHandler {
	return &SSEHandler{db: db}
}

type liveRevenueEvent struct {
	Revenue      int64  `json:"revenue"`
	Transactions int    `json:"transactions"`
	Timestamp    string `json:"timestamp"`
}

// StreamRevenue sends Server-Sent Events with today's revenue and transaction
// count every 10 seconds until the client disconnects.
func (h *SSEHandler) StreamRevenue(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	tenantID := middleware.GetTenantID(r.Context())
	if tenantID == nil {
		response.Error(w, apperror.Forbidden("tenant context required"))
		return
	}

	q := middleware.GetQuerier(r.Context(), h.db)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	send := func() {
		var evt liveRevenueEvent
		err := q.QueryRow(r.Context(), `
			SELECT COALESCE(SUM(total_amount), 0), COUNT(*)
			FROM transactions
			WHERE tenant_id = $1
				AND created_at >= CURRENT_DATE
				AND status = 'completed'
		`, *tenantID).Scan(&evt.Revenue, &evt.Transactions)
		if err != nil {
			slog.Error("SSE: failed to query live revenue", "error", err)
			return
		}
		evt.Timestamp = time.Now().UTC().Format(time.RFC3339)

		data, err := json.Marshal(evt)
		if err != nil {
			slog.Error("SSE: failed to marshal live revenue", "error", err)
			return
		}

		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	// Send immediately on connect, then every tick.
	send()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			send()
		}
	}
}
