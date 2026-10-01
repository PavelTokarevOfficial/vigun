package whisper

import (
	"context"
	"fmt"
	"os/exec"
)

type CommandSource interface {
	CommandFor(context.Context, string) (string, []string, error)
}
type Adapter struct {
	Bin    string
	Source CommandSource
}

func New(bin string, source CommandSource) *Adapter { return &Adapter{Bin: bin, Source: source} }
func (a *Adapter) Transcribe(ctx context.Context, audio, outputBase, modelID string) error {
	model, options, e := a.Source.CommandFor(ctx, modelID)
	if e != nil {
		return fmt.Errorf("whisper settings: %w", e)
	}
	args := []string{"-m", model, "-l", "auto", "-osrt"}
	args = append(args, options...)
	args = append(args, "-of", outputBase, "-f", audio)
	out, e := exec.CommandContext(ctx, a.Bin, args...).CombinedOutput()
	if e != nil {
		return fmt.Errorf("whisper: %w: %s", e, string(out))
	}
	return nil
}
