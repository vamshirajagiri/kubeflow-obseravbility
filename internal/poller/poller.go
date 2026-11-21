package poller

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"kubeflow-pipelines-exporter/internal/config"
	"kubeflow-pipelines-exporter/internal/kfp"
	"kubeflow-pipelines-exporter/internal/state"
)

type Client interface {
	ListRuns(ctx context.Context, namespace string) ([]kfp.Run, int, error)
}

type Poller struct {
	cfg    config.Config
	client Client
	store  *state.Store
	logger *slog.Logger
}

func New(cfg config.Config, client Client, store *state.Store, logger *slog.Logger) *Poller {
	return &Poller{
		cfg:    cfg,
		client: client,
		store:  store,
		logger: logger,
	}
}

func (p *Poller) Run(ctx context.Context) {
	p.poll(ctx)

	ticker := time.NewTicker(p.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

func (p *Poller) poll(ctx context.Context) {
	start := time.Now()
	runs, requests, err := p.listRuns(ctx)
	duration := time.Since(start)
	if err != nil {
		p.store.RecordPollFailure(err, duration, requests)
		p.logger.Error("poll kfp runs", "err", err, "duration", duration)
		return
	}

	counts := countActiveRuns(runs)
	p.store.RecordPollSuccess(counts, len(runs), duration, requests)
	p.logger.Info("poll kfp runs", "runs", len(runs), "active_groups", len(counts), "duration", duration)
}

func (p *Poller) listRuns(ctx context.Context) ([]kfp.Run, int, error) {
	if len(p.cfg.Namespaces) == 0 {
		runs, requests, err := p.client.ListRuns(ctx, "")
		return runs, requests, err
	}

	var allRuns []kfp.Run
	totalRequests := 0
	for _, namespace := range p.cfg.Namespaces {
		runs, requests, err := p.client.ListRuns(ctx, namespace)
		totalRequests += requests
		if err != nil {
			return nil, totalRequests, fmt.Errorf("namespace %s: %w", namespace, err)
		}
		allRuns = append(allRuns, runs...)
	}
	return allRuns, totalRequests, nil
}

func countActiveRuns(runs []kfp.Run) map[string]map[string]int {
	counts := make(map[string]map[string]int)
	for _, run := range runs {
		if !isActiveState(run.State) {
			continue
		}
		namespace := run.Namespace
		if namespace == "" {
			namespace = "unknown"
		}
		if counts[namespace] == nil {
			counts[namespace] = make(map[string]int)
		}
		counts[namespace][run.State]++
	}
	return counts
}

func isActiveState(state string) bool {
	switch state {
	case "PENDING", "RUNNING", "CANCELING", "PAUSED":
		return true
	default:
		return false
	}
}
