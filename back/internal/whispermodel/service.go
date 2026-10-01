package whispermodel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type Status struct {
	State           string `json:"state"`
	DownloadedBytes int64  `json:"downloadedBytes"`
	TotalBytes      int64  `json:"totalBytes"`
	Error           string `json:"error,omitempty"`
}
type Item struct {
	Model
	Installed bool   `json:"installed"`
	Selected  bool   `json:"selected"`
	Status    Status `json:"status"`
}
type Service struct {
	Repo    *Repository
	Client  *http.Client
	Publish func([]byte)
	mu      sync.RWMutex
	states  map[string]Status
}

func New(repo *Repository, publish func([]byte)) *Service {
	return &Service{Repo: repo, Client: &http.Client{Timeout: 0}, Publish: publish, states: map[string]Status{}}
}

func (s *Service) List(ctx context.Context) ([]Item, Settings, error) {
	settings, err := s.Repo.Get(ctx)
	if err != nil {
		return nil, settings, err
	}
	items := make([]Item, 0, len(Catalog))
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, model := range Catalog {
		_, statErr := os.Stat(filepath.Join(s.Repo.Dir, model.Filename))
		status := s.states[model.ID]
		if status.State == "" {
			if statErr == nil {
				status.State = "installed"
			} else {
				status.State = "available"
			}
		}
		items = append(items, Item{Model: model, Installed: statErr == nil, Selected: settings.ActiveModel == model.ID, Status: status})
	}
	return items, settings, nil
}

func (s *Service) StartDownload(id string) error {
	model, err := Find(id)
	if err != nil {
		return err
	}
	s.mu.Lock()
	if s.states[id].State == "downloading" {
		s.mu.Unlock()
		return fmt.Errorf("model is already downloading")
	}
	s.states[id] = Status{State: "downloading", TotalBytes: model.SizeBytes}
	s.mu.Unlock()
	go s.download(model)
	s.notify()
	return nil
}

func (s *Service) download(model Model) {
	if err := os.MkdirAll(s.Repo.Dir, 0755); err != nil {
		s.fail(model.ID, err)
		return
	}
	final := filepath.Join(s.Repo.Dir, model.Filename)
	partial := final + ".part"
	if _, err := os.Stat(final); err == nil {
		s.set(model.ID, Status{State: "installed", TotalBytes: model.SizeBytes, DownloadedBytes: model.SizeBytes})
		return
	}
	_ = os.Remove(partial)
	var disk syscall.Statfs_t
	if err := syscall.Statfs(s.Repo.Dir, &disk); err == nil && int64(disk.Bavail)*int64(disk.Bsize) < model.SizeBytes+(100<<20) {
		s.fail(model.ID, fmt.Errorf("недостаточно свободного места"))
		return
	}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, downloadURL(model), nil)
	resp, err := s.Client.Do(req)
	if err != nil {
		s.fail(model.ID, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		s.fail(model.ID, fmt.Errorf("download returned %s", resp.Status))
		return
	}
	total := resp.ContentLength
	if total <= 0 {
		total = model.SizeBytes
	}
	f, err := os.OpenFile(partial, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		s.fail(model.ID, err)
		return
	}
	w := &progressWriter{service: s, id: model.ID, total: total, writer: f}
	_, copyErr := io.Copy(w, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(partial)
		s.fail(model.ID, copyErr)
		return
	}
	if closeErr != nil {
		_ = os.Remove(partial)
		s.fail(model.ID, closeErr)
		return
	}
	info, err := os.Stat(partial)
	if err != nil || info.Size() != total {
		_ = os.Remove(partial)
		s.fail(model.ID, fmt.Errorf("скачивание завершилось не полностью"))
		return
	}
	if err = os.Rename(partial, final); err != nil {
		_ = os.Remove(partial)
		s.fail(model.ID, err)
		return
	}
	s.set(model.ID, Status{State: "installed", DownloadedBytes: total, TotalBytes: total})
}

type progressWriter struct {
	service *Service
	id      string
	total   int64
	writer  io.Writer
	written int64
	last    time.Time
}

func (w *progressWriter) Write(p []byte) (int, error) {
	n, e := w.writer.Write(p)
	w.written += int64(n)
	if time.Since(w.last) > 500*time.Millisecond {
		w.last = time.Now()
		w.service.set(w.id, Status{State: "downloading", DownloadedBytes: w.written, TotalBytes: w.total})
	}
	return n, e
}
func (s *Service) set(id string, state Status) {
	s.mu.Lock()
	s.states[id] = state
	s.mu.Unlock()
	s.notify()
}
func (s *Service) fail(id string, err error) { s.set(id, Status{State: "error", Error: err.Error()}) }
func (s *Service) notify() {
	if s.Publish != nil {
		b, _ := json.Marshal(map[string]string{"scope": "settings", "type": "whisper_models_changed"})
		s.Publish(b)
	}
}

func (s *Service) Select(ctx context.Context, id string) error {
	path, err := s.Repo.ModelPath(id)
	if err != nil {
		return err
	}
	if _, err = os.Stat(path); err != nil {
		return fmt.Errorf("model is not installed")
	}
	_, err = s.Repo.DB.Exec(ctx, `UPDATE whisper_settings SET active_model=$1,updated_at=now() WHERE singleton=true`, id)
	if err == nil {
		s.notify()
	}
	return err
}

func (s *Service) Installed(id string) bool {
	path, err := s.Repo.ModelPath(id)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}
func (s *Service) Delete(ctx context.Context, id string) error {
	settings, err := s.Repo.Get(ctx)
	if err != nil {
		return err
	}
	if settings.ActiveModel == id {
		return fmt.Errorf("active model cannot be deleted")
	}
	model, err := Find(id)
	if err != nil {
		return err
	}
	err = os.Remove(filepath.Join(s.Repo.Dir, model.Filename))
	if os.IsNotExist(err) {
		err = nil
	}
	if err == nil {
		s.mu.Lock()
		delete(s.states, id)
		s.mu.Unlock()
		s.notify()
	}
	return err
}
func (s *Service) Update(ctx context.Context, in Settings) error {
	if _, err := Find(in.ActiveModel); err != nil {
		return err
	}
	if in.Preset != "fast" && in.Preset != "balanced" && in.Preset != "maximum" && in.Preset != "subtitles" {
		return fmt.Errorf("invalid preset")
	}
	if in.BeamSize != nil && (*in.BeamSize < 1 || *in.BeamSize > 20) {
		return fmt.Errorf("beam size must be from 1 to 20")
	}
	if in.Temperature != nil && (*in.Temperature < 0 || *in.Temperature > 1) {
		return fmt.Errorf("temperature must be from 0 to 1")
	}
	if in.MaxSegmentLength != nil && (*in.MaxSegmentLength < 1 || *in.MaxSegmentLength > 200) {
		return fmt.Errorf("maximum segment length must be from 1 to 200")
	}
	_, err := s.Repo.DB.Exec(ctx, `UPDATE whisper_settings SET preset=$1,beam_size=$2,temperature=$3,max_segment_length=$4,split_on_word=$5,initial_prompt=$6,updated_at=now() WHERE singleton=true`, in.Preset, in.BeamSize, in.Temperature, in.MaxSegmentLength, in.SplitOnWord, in.InitialPrompt)
	if err == nil {
		s.notify()
	}
	return err
}
