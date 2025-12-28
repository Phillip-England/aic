package aic

import (
	"fmt"

	"github.com/go-vgo/robotgo"
)

// Sequence "1": print current mouse coordinates.
type MouseCoordsSequence struct{}

func (MouseCoordsSequence) Key() string  { return "1" }
func (MouseCoordsSequence) Name() string { return "Print mouse coordinates" }

func (MouseCoordsSequence) Run(ctx SeqContext) error {
	x, y := robotgo.GetMousePos()
	// Print to terminal (Err is typically where aic logs status).
	_, _ = fmt.Fprintf(ctx.Err, "[seq ';1] mouse=(%d,%d)\n", x, y)
	return nil
}
