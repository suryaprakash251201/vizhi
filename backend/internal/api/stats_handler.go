package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"vizhi/backend/internal/monitor"
)

type StatsHandler struct {
	mon *monitor.Monitor
}

func NewStatsHandler(mon *monitor.Monitor) *StatsHandler {
	return &StatsHandler{mon: mon}
}

func (h *StatsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	stats, err := h.mon.Gather(ctx)
	if err != nil {
		http.Error(w, `{"error":"failed to gather stats"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
