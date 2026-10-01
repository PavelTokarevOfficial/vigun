package whispermodel

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Settings struct {
	ActiveModel      string   `json:"activeModel"`
	Preset           string   `json:"preset"`
	BeamSize         *int     `json:"beamSize"`
	Temperature      *float64 `json:"temperature"`
	MaxSegmentLength *int     `json:"maxSegmentLength"`
	SplitOnWord      *bool    `json:"splitOnWord"`
	InitialPrompt    string   `json:"initialPrompt"`
}

type Repository struct {
	DB  *pgxpool.Pool
	Dir string
}

func (r *Repository) Get(ctx context.Context) (Settings, error) {
	var s Settings
	err := r.DB.QueryRow(ctx, `SELECT active_model,preset,beam_size,temperature,max_segment_length,split_on_word,initial_prompt FROM whisper_settings WHERE singleton=true`).Scan(&s.ActiveModel, &s.Preset, &s.BeamSize, &s.Temperature, &s.MaxSegmentLength, &s.SplitOnWord, &s.InitialPrompt)
	return s, err
}

func (r *Repository) ModelPath(id string) (string, error) {
	m, err := Find(id)
	if err != nil {
		return "", err
	}
	return filepath.Join(r.Dir, m.Filename), nil
}

func (r *Repository) Command(ctx context.Context) (string, []string, error) {
	return r.CommandFor(ctx, "")
}

func (r *Repository) CommandFor(ctx context.Context, modelID string) (string, []string, error) {
	s, err := r.Get(ctx)
	if err != nil {
		return "", nil, err
	}
	if modelID == "" {
		modelID = s.ActiveModel
	}
	path, err := r.ModelPath(modelID)
	if err != nil {
		return "", nil, err
	}
	options := presetOptions(s.Preset)
	if s.BeamSize != nil {
		options.beamSize = s.BeamSize
	}
	if s.MaxSegmentLength != nil {
		options.maxLength = s.MaxSegmentLength
	}
	if s.SplitOnWord != nil {
		options.splitOnWord = *s.SplitOnWord
	}
	args := options.args()
	if s.Temperature != nil {
		args = append(args, "-tp", fmt.Sprint(*s.Temperature))
	}
	if strings.TrimSpace(s.InitialPrompt) != "" {
		args = append(args, "--prompt", s.InitialPrompt)
	}
	return path, args, nil
}

type commandOptions struct {
	beamSize, bestOf, maxLength *int
	noFallback, splitOnWord     bool
}

func intPtr(value int) *int { return &value }
func presetOptions(preset string) commandOptions {
	switch preset {
	case "fast":
		return commandOptions{beamSize: intPtr(1), bestOf: intPtr(1), noFallback: true}
	case "maximum":
		return commandOptions{beamSize: intPtr(8)}
	case "subtitles":
		return commandOptions{maxLength: intPtr(42), splitOnWord: true}
	default:
		return commandOptions{maxLength: intPtr(10), splitOnWord: true}
	}
}
func (o commandOptions) args() []string {
	var args []string
	if o.beamSize != nil {
		args = append(args, "-bs", fmt.Sprint(*o.beamSize))
	}
	if o.bestOf != nil {
		args = append(args, "-bo", fmt.Sprint(*o.bestOf))
	}
	if o.noFallback {
		args = append(args, "-nf")
	}
	if o.maxLength != nil {
		args = append(args, "-ml", fmt.Sprint(*o.maxLength))
	}
	if o.splitOnWord {
		args = append(args, "-sow")
	}
	return args
}
