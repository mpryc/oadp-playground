package demystifier

import "time"

const (
	Failed  = "FAILED"
	Passed  = "PASSED"
	Timeout = "TIMEOUT"
)

type EventStatus struct {
	Status string
}

func (s *EventStatus) SetFailed() {
	s.Status = Failed
}
func (s *EventStatus) SetPassing() {
	s.Status = Passed
}
func (s *EventStatus) SetTimeout() {
	s.Status = Timeout
}

// Event is for example Backup or Restore
type EventData struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Status    EventStatus
	Logs      []string
}

// Attempt is for a single Test run that may include
// multiple Events
type AttemptData struct {
	AttemptNo int
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Status    EventStatus // Don't yet know if it is better to be here or in the EventData
	Logs      []string
	Events    []EventData
}

// IndividualTestRunData may consists of many attempts, each attempt
// is run of the same test, but may lead to different
// results or failures
type IndividualTestRunData struct {
	Name      string
	ShortName string
	Attempt   []AttemptData
}

// This is representation of full run, it may not have tests itself
// but w want to store full log
type TestRunData struct {
	FullLogs string
	TestRun  []IndividualTestRunData
}
