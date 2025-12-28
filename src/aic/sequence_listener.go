package aic

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	hook "github.com/robotn/gohook"
)

type SequenceListener struct {
	mgr         *SequenceManager
	leaders     []rune
	stepTimeout time.Duration

	debugMu  sync.Mutex
	debugf   func(format string, args ...any)
	debugAll bool // log every key event + stage transitions

	lastRaw   uint16
	lastRawAt time.Time

	mu    sync.Mutex
	stage int       // 0=start, k=matched k chars of leaders
	last  time.Time // last relevant key time
}

func NewSequenceListener(mgr *SequenceManager) *SequenceListener {
	return &SequenceListener{
		mgr:         mgr,
		leaders:     []rune{' ', '\'', ';'},
		stepTimeout: 2500 * time.Millisecond,
	}
}

func (l *SequenceListener) EnableDebug(f func(format string, args ...any)) {
	l.debugMu.Lock()
	defer l.debugMu.Unlock()
	l.debugf = f
}

func (l *SequenceListener) EnableDebugAllKeys(enable bool) {
	l.debugMu.Lock()
	defer l.debugMu.Unlock()
	l.debugAll = enable
}

func (l *SequenceListener) dbg(format string, args ...any) {
	l.debugMu.Lock()
	f := l.debugf
	l.debugMu.Unlock()
	if f == nil {
		return
	}
	f(format, args...)
}

func (l *SequenceListener) debugAllEnabled() bool {
	l.debugMu.Lock()
	defer l.debugMu.Unlock()
	return l.debugAll
}

func (l *SequenceListener) Start(run func(key string) error) (stop func(), err error) {
	if l.mgr == nil {
		return nil, fmt.Errorf("sequence listener requires SequenceManager")
	}

	evChan := hook.Start()
	done := make(chan struct{})

	go func() {
		defer close(done)

		for ev := range evChan {
			if ev.Kind != hook.KeyDown {
				continue
			}

			raw := uint16(ev.Rawcode)
			now := time.Now()

			// de-dupe repeated keydown bursts
			if raw != 0 && raw == l.lastRaw && !l.lastRawAt.IsZero() && now.Sub(l.lastRawAt) < 60*time.Millisecond {
				continue
			}
			l.lastRaw = raw
			l.lastRawAt = now

			ch, ok, src := eventRuneDebug(ev)

			if l.debugAllEnabled() {
				if ok {
					l.dbg("[seq dbg] ev: kind=%v keychar=%d raw=%d => %q (U+%04X) src=%s stage=%d\n",
						ev.Kind, ev.Keychar, ev.Rawcode, ch, ch, src, l.stageSnapshot())
				} else {
					l.dbg("[seq dbg] ev: kind=%v keychar=%d raw=%d => <undecodable> src=%s stage=%d\n",
						ev.Kind, ev.Keychar, ev.Rawcode, src, l.stageSnapshot())
				}
			} else {
				if ok && (l.stageSnapshot() > 0 || ch == l.leaders[0]) {
					l.dbg("[seq dbg] ev: kind=%v keychar=%d raw=%d => %q (%s)\n", ev.Kind, ev.Keychar, ev.Rawcode, ch, src)
				}
				if !ok && l.stageSnapshot() > 0 {
					l.dbg("[seq dbg] event undecodable: kind=%v keychar=%d raw=%d\n", ev.Kind, ev.Keychar, ev.Rawcode)
				}
			}

			if !ok {
				continue
			}

			cmdKey, fire := l.feed(ch)
			if !fire {
				continue
			}

			cmdKey = strings.ToLower(strings.TrimSpace(cmdKey))
			if len(cmdKey) != 1 {
				if l.debugAllEnabled() {
					l.dbg("[seq dbg] fire ignored: cmdKey=%q (need exactly 1 char)\n", cmdKey)
				}
				continue
			}

			if l.debugAllEnabled() {
				l.dbg("[seq dbg] FIRE cmd=%q\n", cmdKey)
			}

			_ = run(cmdKey)
		}
	}()

	return func() {
		hook.End()
		<-done
	}, nil
}

func (l *SequenceListener) stageSnapshot() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.stage
}

// eventRuneDebug tries hard to turn a keydown event into a single printable rune.
// On macOS, SPACE is frequently Rawcode=49 but RawcodetoKeychar() may return "".
// We add a rawcode fallback so leader sequences don't randomly fail when typed fast.
func eventRuneDebug(ev hook.Event) (rune, bool, string) {
	// Prefer Keychar if it’s usable
	if ev.Keychar != 0 && ev.Keychar != hook.CharUndefined {
		r := rune(ev.Keychar)
		if r < 32 {
			return 0, false, "keychar<32"
		}
		return r, true, "keychar"
	}

	// Next try library mapping
	s := hook.RawcodetoKeychar(ev.Rawcode)
	s = strings.TrimSpace(s)
	if s == "" {
		// Rawcode fallback (important on macOS)
		if runtime.GOOS == "darwin" {
			switch uint16(ev.Rawcode) {
			case 49:
				return ' ', true, "rawcode(darwin:space=49)"
			case 41:
				return ';', true, "rawcode(darwin:semicolon=41)"
			case 39:
				return '\'', true, "rawcode(darwin:apostrophe=39)"
			}
		}
		return 0, false, "rawcode->empty"
	}

	rs := []rune(s)
	if len(rs) == 1 {
		r := rs[0]
		if r < 32 {
			return 0, false, "rawcode->rune<32"
		}
		return r, true, "rawcode"
	}

	// Named keys sometimes come back as words
	switch strings.ToLower(s) {
	case "space":
		return ' ', true, "rawcode(space)"
	case "semicolon":
		return ';', true, "rawcode(semicolon)"
	case "quote", "apostrophe":
		return '\'', true, "rawcode(apostrophe)"
	default:
		return 0, false, "rawcode->named(" + s + ")"
	}
}

func (l *SequenceListener) feed(ch rune) (string, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if !l.last.IsZero() && now.Sub(l.last) > l.stepTimeout {
		if l.stage != 0 {
			l.dbg("[seq dbg] timeout; resetting stage from %d\n", l.stage)
		}
		l.stage = 0
	}
	l.last = now

	if len(l.leaders) == 0 {
		return string(ch), true
	}

	if l.stage < len(l.leaders) {
		target := l.leaders[l.stage]
		if l.debugAllEnabled() {
			l.dbg("[seq dbg] feed: ch=%q (U+%04X) stage=%d expecting=%q (U+%04X)\n",
				ch, ch, l.stage, target, target)
		}

		if l.stage > 0 && ch == l.leaders[l.stage-1] {
			l.dbg("[seq dbg] ignored repeat leader char at stage %d: %q (U+%04X)\n", l.stage, ch, ch)
			return "", false
		}

		if ch == target {
			l.stage++
			l.dbg("[seq dbg] matched leader %d/%d: %q\n", l.stage, len(l.leaders), ch)
			if l.stage == len(l.leaders) {
				l.dbg("[seq dbg] leader registered: %q\n", string(l.leaders))
			}
			return "", false
		}

		if ch == l.leaders[0] {
			l.stage = 1
			l.dbg("[seq dbg] restarted leader 1/%d: %q\n", len(l.leaders), ch)
			return "", false
		}

		if l.stage != 0 {
			l.dbg("[seq dbg] leader broken at stage %d (expected %q) by %q; reset\n", l.stage, target, ch)
		}
		l.stage = 0
		return "", false
	}

	// Stage complete: next key is the command key
	l.stage = 0
	return string(ch), true
}
