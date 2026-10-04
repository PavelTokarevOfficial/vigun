package browser

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// RodDownloader owns all Twitch-page selectors and can be replaced without touching pipeline code.
type RodDownloader struct {
	headless bool
	bin      string
}

func NewRodDownloader(headless bool, bin string) *RodDownloader {
	return &RodDownloader{headless: headless, bin: bin}
}

type downloadedFile struct {
	*os.File
	dir string
}

func (f *downloadedFile) Close() error {
	err := f.File.Close()
	removeErr := os.RemoveAll(f.dir)
	if err != nil {
		return err
	}
	return removeErr
}

func (d *RodDownloader) Download(ctx context.Context, clipURL string) (io.ReadCloser, string, error) {
	l := launcher.New().Headless(d.headless)
	if d.bin != "" {
		l.Bin(d.bin)
	}
	u, e := l.Launch()
	if e != nil {
		return nil, "", fmt.Errorf("launch Chromium: %w", e)
	}
	b := rod.New().ControlURL(u)
	if e = b.Connect(); e != nil {
		return nil, "", fmt.Errorf("connect Chromium: %w", e)
	}
	defer func() { _ = b.Close() }()
	p, e := b.Context(ctx).Page(proto.TargetCreateTarget{})
	if e != nil {
		return nil, "", e
	}
	if e = p.Navigate(clipURL); e != nil {
		return nil, "", e
	}
	if e = p.WaitLoad(); e != nil {
		return nil, "", e
	}

	share, e := p.Timeout(30 * time.Second).Element(`button[aria-label^="Поделиться клипом"],button[aria-label^="Share clip"],button[aria-label^="Share Clip"]`)
	if e != nil {
		share, e = p.Timeout(10*time.Second).ElementR("button", `^\s*(Поделиться|Share|share)\s*$`)
	}
	if e != nil {
		return nil, "", fmt.Errorf("find Twitch share button: %w", e)
	}
	if e = share.ScrollIntoView(); e != nil {
		return nil, "", fmt.Errorf("scroll to Twitch share button: %w", e)
	}
	if e = share.Click(proto.InputMouseButtonLeft, 1); e != nil {
		return nil, "", fmt.Errorf("open Twitch share menu: %w", e)
	}

	download, e := p.Timeout(20 * time.Second).Element(`a[href*="/landscape/"][href*="/source/index.mp4"]`)
	if e != nil {
		download, e = p.Timeout(10*time.Second).ElementR("a", `^\s*(Скачать альбомную версию|Download landscape version|Download Landscape)\s*$`)
	}
	if e != nil {
		return nil, "", fmt.Errorf("find Twitch landscape download: %w", e)
	}
	dir, e := os.MkdirTemp("", "finde-twitch-download-")
	if e != nil {
		return nil, "", fmt.Errorf("create Twitch download directory: %w", e)
	}
	wait := b.Context(ctx).WaitDownload(dir)
	if e = download.Click(proto.InputMouseButtonLeft, 1); e != nil {
		_ = os.RemoveAll(dir)
		return nil, "", fmt.Errorf("start Twitch landscape download: %w", e)
	}
	info := wait()
	if info == nil {
		_ = os.RemoveAll(dir)
		return nil, "", fmt.Errorf("Twitch landscape download did not start")
	}
	file, e := os.Open(filepath.Join(dir, info.GUID))
	if e != nil {
		_ = os.RemoveAll(dir)
		return nil, "", fmt.Errorf("open Twitch landscape download: %w", e)
	}
	return &downloadedFile{File: file, dir: dir}, "video/mp4", nil
}
