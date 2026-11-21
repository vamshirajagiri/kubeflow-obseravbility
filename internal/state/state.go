package state

import (
	"sync"
	"time"
)

type Snapshot struct {
	StartTime         time.Time
	LastPollTime      time.Time
	LastPollSuccess   bool
	LastPollError     string
	APIRequests       map[string]map[string]uint64
	RunsCurrent       map[string]map[string]int
	LastPollDuration  time.Duration
	LastPollRunCount  int
	LastPollNamespace string
}

type Store struct {
	mu       sync.RWMutex
	snapshot Snapshot
}

func NewStore(startTime time.Time) *Store {
	return &Store{
		snapshot: Snapshot{
			StartTime:   startTime,
			APIRequests: make(map[string]map[string]uint64),
			RunsCurrent: make(map[string]map[string]int),
		},
	}
}

func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return cloneSnapshot(s.snapshot)
}

func (s *Store) RecordPollSuccess(runs map[string]map[string]int, runCount int, duration time.Duration, requests int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.snapshot.LastPollTime = time.Now()
	s.snapshot.LastPollSuccess = true
	s.snapshot.LastPollError = ""
	s.snapshot.LastPollDuration = duration
	s.snapshot.LastPollRunCount = runCount
	s.snapshot.RunsCurrent = cloneRuns(runs)
	s.addAPIRequests("runs", "success", uint64(requests))
}

func (s *Store) RecordPollFailure(err error, duration time.Duration, requests int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.snapshot.LastPollTime = time.Now()
	s.snapshot.LastPollSuccess = false
	s.snapshot.LastPollDuration = duration
	if err != nil {
		s.snapshot.LastPollError = err.Error()
	}
	if requests <= 0 {
		requests = 1
	}
	s.addAPIRequests("runs", "error", uint64(requests))
}

func (s *Store) addAPIRequests(source, result string, value uint64) {
	if s.snapshot.APIRequests[source] == nil {
		s.snapshot.APIRequests[source] = make(map[string]uint64)
	}
	s.snapshot.APIRequests[source][result] += value
}

func cloneSnapshot(snapshot Snapshot) Snapshot {
	snapshot.APIRequests = cloneRequests(snapshot.APIRequests)
	snapshot.RunsCurrent = cloneRuns(snapshot.RunsCurrent)
	return snapshot
}

func cloneRequests(in map[string]map[string]uint64) map[string]map[string]uint64 {
	out := make(map[string]map[string]uint64, len(in))
	for source, results := range in {
		out[source] = make(map[string]uint64, len(results))
		for result, value := range results {
			out[source][result] = value
		}
	}
	return out
}

func cloneRuns(in map[string]map[string]int) map[string]map[string]int {
	out := make(map[string]map[string]int, len(in))
	for namespace, states := range in {
		out[namespace] = make(map[string]int, len(states))
		for state, value := range states {
			out[namespace][state] = value
		}
	}
	return out
}
