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

	AlertPriority  int
	AlertYieldFunc func(alert Alert)
	Alerts         Model
)

const (
	None AlertPriority = iota
	Info
	Warning
	Error
)

// Model methods

func (m *Model) Alerts() *Alerts { return (*Alerts)(m) }

// Alerts methods

func (m *Alerts) Iterate(yield AlertYieldFunc) {
	for _, a := range m.alerts {
		yield(a)
	}
}

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

func (m *Alerts) Add(message string, priority AlertPriority) {
	m.AddAlert(Alert{
		Priority: priority,
		Message:  message,
		Duration: defaultAlertDuration,
	})
}

func (m *Alerts) AddNamed(name, message string, priority AlertPriority) {
	m.AddAlert(Alert{
		Name:     name,
		Priority: priority,
		Message:  message,
		Duration: defaultAlertDuration,
	})
}

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
	m.alerts = append(m.alerts, x.(Alert))
}

func (m *Alerts) Pop() any {
	old := m.alerts
	n := len(old)
	x := old[n-1]
	m.alerts = old[0 : n-1]
	return x
}

func (m Alerts) Len() int           { return len(m.alerts) }
func (m Alerts) Less(i, j int) bool { return m.alerts[i].Priority < m.alerts[j].Priority }
func (m Alerts) Swap(i, j int)      { m.alerts[i], m.alerts[j] = m.alerts[j], m.alerts[i] }
