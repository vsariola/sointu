package tracker

import (
	"container/heap"
	"time"
)

const alertSpeed = 10 // units: fadeLevels per second
const defaultAlertDuration = time.Second * 3

type (
	Alert struct {
		Name      string
		Priority  AlertPriority
		Message   string
		Duration  time.Duration
		FadeLevel float64
	}

	AlertPriority int
)

const (
	None AlertPriority = iota
	Info
	Warning
	Error
)

// Alerts returns the Alerts model from the main Model, used to manage alerts.
func (m *Model) Alerts() *Alerts { return (*Alerts)(m) }

type Alerts Model

// Iterate through the alerts.
func (m *Alerts) Iterate(yield func(index int, alert Alert) bool) {
	for i, a := range m.alerts {
		if !yield(i, a) {
			break
		}
	}
}

// Update the alerts, reducing their duration and updating their fade levels,
// given the elapsed time d.
func (m *Alerts) Update(d time.Duration) (animating bool) {
	for i := len(m.alerts) - 1; i >= 0; i-- {
		if m.alerts[i].Duration >= d {
			m.alerts[i].Duration -= d
			if m.alerts[i].FadeLevel < 1 {
				animating = true
				m.alerts[i].FadeLevel += float64(alertSpeed*d) / float64(time.Second)
				if m.alerts[i].FadeLevel > 1 {
					m.alerts[i].FadeLevel = 1
				}
			}
		} else {
			m.alerts[i].Duration = 0
			m.alerts[i].FadeLevel -= float64(alertSpeed*d) / float64(time.Second)
			animating = true
			if m.alerts[i].FadeLevel < 0 {
				heap.Remove(m, i)
			}
		}
	}
	return
}

// Add a new alert with the given message and priority.
func (m *Alerts) Add(message string, priority AlertPriority) {
	m.AddAlert(Alert{
		Priority: priority,
		Message:  message,
		Duration: defaultAlertDuration,
	})
}

// AddNamed adds a new alert with the given name, message, and priority.
func (m *Alerts) AddNamed(name, message string, priority AlertPriority) {
	m.AddAlert(Alert{
		Name:     name,
		Priority: priority,
		Message:  message,
		Duration: defaultAlertDuration,
	})
}

// ClearNamed clears the alert with the given name.
func (m *Alerts) ClearNamed(name string) {
	for i := range m.alerts {
		if n := m.alerts[i].Name; n != "" && n == name {
			m.alerts[i].Duration = 0
			return
		}
	}
}

// AddAlert adds or updates an alert.
func (m *Alerts) AddAlert(a Alert) {
	for i := range m.alerts {
		if n := m.alerts[i].Name; n != "" && n == a.Name {
			a.FadeLevel = m.alerts[i].FadeLevel
			m.alerts[i] = a
			heap.Fix(m, i)
			return
		}
	}
	heap.Push(m, a)
}

func (m *Alerts) Push(x any) {
	if _, ok := x.(Alert); !ok {
		panic("invalid type for Alerts.Push, expected Alert")
	}
	m.alerts = append(m.alerts, x.(Alert))
}

func (m *Alerts) Pop() any {
	n := len(m.alerts)
	last := m.alerts[n-1]
	m.alerts = m.alerts[:n-1]
	return last
}

func (m Alerts) Len() int           { return len(m.alerts) }
func (m Alerts) Less(i, j int) bool { return m.alerts[i].Priority < m.alerts[j].Priority }
func (m Alerts) Swap(i, j int)      { m.alerts[i], m.alerts[j] = m.alerts[j], m.alerts[i] }
