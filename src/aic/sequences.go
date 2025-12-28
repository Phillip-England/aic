package aic

import "io"

// SeqContext is the runtime context available to sequences.
type SeqContext struct {
	Out io.Writer
	Err io.Writer
}

// Sequence is a single leader-triggered command.
//
// A sequence is invoked by typing the leader "';" followed by Sequence.Key().
// Example: leader + "1" calls the sequence whose Key() == "1".
type Sequence interface {
	// Key is the 1-char command key after the leader (digit/letter).
	Key() string

	// Name is a human-readable name for help/logs.
	Name() string

	// Run executes the sequence.
	Run(ctx SeqContext) error
}
