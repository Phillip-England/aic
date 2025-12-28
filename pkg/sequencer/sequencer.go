package sequencer

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/go-vgo/robotgo"
	hook "github.com/robotn/gohook"
)

type SeqContext struct {
	Out io.Writer
	Err io.Writer
}

type Sequence interface {
	Key() string
	Run(ctx SeqContext) error
}

type Manager struct {
	mu  sync.RWMutex
	seq map[string]Sequence
}

func NewManager() *Manager {
	return &Manager{seq: make(map[string]Sequence)}
}

func (m *Manager) Register(s Sequence) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := strings.ToLower(strings.TrimSpace(s.Key()))
	m.seq[key] = s
}

func (m *Manager) Get(k string) (Sequence, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.seq[strings.ToLower(k)]
	return s, ok
}

// --- Built-in Sequences ---

type MouseCoordsSeq struct{}

func (MouseCoordsSeq) Key() string { return "1" }
func (MouseCoordsSeq) Run(ctx SeqContext) error {
	x, y := robotgo.GetMousePos()
	fmt.Fprintf(ctx.Err, "\n[AIC] Mouse Coords: %d, %d\n", x, y)
	return nil
}

// --- Listener ---

type Listener struct {
	mgr         *Manager
	leaders     []rune
	stage       int
	last        time.Time
	stepTimeout time.Duration
	mu          sync.Mutex
}

func NewListener(m *Manager) *Listener {
	return &Listener{
		mgr:         m,
		leaders:     []rune{' ', '\'', ';'}, // SPACE -> ' -> ;
		stepTimeout: 3000 * time.Millisecond,
	}
}

func (l *Listener) Start(errOut io.Writer) (func(), error) {
	// Start the hook
	evChan := hook.Start()
	done := make(chan struct{})

	go func() {
		defer close(done)
		for ev := range evChan {
			// Only care about KeyDown
			if ev.Kind != hook.KeyDown {
				continue
			}

			// Convert event to rune
			ch := l.eventToRune(ev)
			if ch == 0 {
				continue
			}

			// Feed into state machine
			cmdKey, fire := l.feed(ch)

			// If state machine says fire, execute command
			if fire {
				if seq, ok := l.mgr.Get(cmdKey); ok {
					fmt.Fprintf(errOut, "[AIC] Sequence Triggered: %s\n", seq.Key())
					// Run in goroutine to not block the hook
					go seq.Run(SeqContext{Err: errOut})
				} else {
					fmt.Fprintf(errOut, "[AIC] Unknown Sequence Command: %s\n", cmdKey)
				}
			}
		}
	}()

	return func() {
		hook.End()
		<-done
	}, nil
}

func (l *Listener) eventToRune(ev hook.Event) rune {
	// 1. Try Keychar first (most reliable for printable chars)
	if ev.Keychar != 0 {
		return rune(ev.Keychar)
	}

	// 2. Fallback to Rawcode for specific keys if Keychar is missing
	// Mac/Linux common codes (approximate)
	switch ev.Rawcode {
	case 49, 32: // Space
		return ' '
	case 39, 40: // ' or similar
		return '\''
	case 41, 47: // ; or similar
		return ';'
	}

	// 3. Try robotgo map
	s := hook.RawcodetoKeychar(ev.Rawcode)
	if s == "space" { return ' ' }
	if len(s) == 1 {
		return []rune(s)[0]
	}
	
	return 0
}

func (l *Listener) feed(ch rune) (string, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	// Reset if timeout
	if !l.last.IsZero() && now.Sub(l.last) > l.stepTimeout {
		if l.stage > 0 {
			// Debug output only when resetting active state
			// fmt.Printf("Timeout, resetting stage\n")
		}
		l.stage = 0
	}
	l.last = now

	// TARGET logic
	// If stage == len(leaders), we are waiting for the final command key
	if l.stage == len(l.leaders) {
		l.stage = 0 // Reset
		return string(ch), true
	}

	// MATCH logic
	target := l.leaders[l.stage]
	
	if ch == target {
		l.stage++
		// fmt.Printf("Matched Stage %d/%d: %c\n", l.stage, len(l.leaders), ch)
		return "", false
	}

	// RESTART logic
	// If we typed SPACE but we were expecting ';', we should reset to stage 1 (SPACE matched)
	if ch == l.leaders[0] {
		l.stage = 1
		// fmt.Printf("Restarted Stage 1\n")
		return "", false
	}

	// RESET logic
	if l.stage > 0 {
		// fmt.Printf("Broken Sequence at stage %d\n", l.stage)
	}
	l.stage = 0
	return "", false
}