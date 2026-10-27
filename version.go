package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	"time"

	"assetgoblin/utils"

	"charm.land/lipgloss/v2"
	"github.com/Masterminds/semver/v3"
	"github.com/charmbracelet/x/term"
	"github.com/creativeprojects/go-selfupdate"
)

// Version is the release tag, set with -ldflags at build time.
var Version = "development"

// BuildTime is the build timestamp, set with -ldflags.
var BuildTime = "unknown"

// GitCommit is the source revision, set with -ldflags.
var GitCommit = "unknown"

// newUpdater matches the archive names produced by GoReleaser and requires SHA-256 checksums.
func newUpdater(goos, goarch string, out io.Writer) (*selfupdate.Updater, error) {
	arch := map[string]string{"amd64": "x64", "386": "x86", "arm64": "arm64"}[goarch]
	if arch == "" {
		return nil, fmt.Errorf("no release archive for architecture %q", goarch)
	}
	osName, extension := goos, ".tar.gz"
	if goos == "darwin" {
		osName = "macOS"
	}
	if goos == "windows" {
		extension = ".zip"
	}
	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
	if err != nil {
		return nil, err
	}
	var downloadSource selfupdate.Source = source
	if out != nil {
		downloadSource = &updateSource{Source: source, out: out}
	}
	return selfupdate.NewUpdater(selfupdate.Config{
		Source: downloadSource, OS: goos, Arch: goarch,
		Filters:   []string{"(?i)^AssetGoblin_v?[^_]+_" + regexp.QuoteMeta(osName+"_"+arch+extension) + "$"},
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	})
}

func detectLatest(updater *selfupdate.Updater) (*selfupdate.Release, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	latest, found, err := updater.DetectLatest(ctx, selfupdate.NewRepositorySlug("sbolch", "AssetGoblin"))
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("no compatible release available")
	}
	return latest, nil
}

func getLatestRelease() (*selfupdate.Release, error) {
	updater, err := newUpdater(runtime.GOOS, runtime.GOARCH, nil)
	if err != nil {
		return nil, err
	}
	return detectLatest(updater)
}

// Development builds can update; released builds never downgrade.
func updateAvailable(latest *selfupdate.Release) bool {
	current, err := semver.NewVersion(Version)
	return err != nil || latest.GreaterThan(current.String())
}

// update replaces the installed executable after validating the complete download.
func update() error {
	updater, err := newUpdater(runtime.GOOS, runtime.GOARCH, os.Stdout)
	if err != nil {
		return err
	}
	latest, err := detectLatest(updater)
	if err != nil {
		return err
	}
	if !updateAvailable(latest) {
		printSuccess("Already up to date.")
		return nil
	}
	target, err := selfupdate.ExecutablePath()
	if err != nil {
		return err
	}
	lipgloss.Fprintln(os.Stdout, cliHeading.Render("Updating "+Version+" → v"+latest.Version()))
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := updater.UpdateTo(ctx, latest, target); err != nil {
		return err
	}
	printSuccess("Update finished successfully.")
	width := 80
	if columns, _, err := term.GetSize(os.Stdout.Fd()); err == nil && columns > 0 {
		width = min(columns, width)
	}
	printReleaseNotes(os.Stdout, latest.ReleaseNotes, width)
	return nil
}

// updateSource adds progress to archive downloads, leaving checksum downloads quiet.
type updateSource struct {
	selfupdate.Source
	out io.Writer
}

func (s *updateSource) DownloadReleaseAsset(ctx context.Context, release *selfupdate.Release, assetID int64) (io.ReadCloser, error) {
	reader, err := s.Source.DownloadReleaseAsset(ctx, release, assetID)
	if err != nil || assetID != release.AssetID {
		return reader, err
	}
	interactive := s.out == os.Stdout && term.IsTerminal(os.Stdout.Fd()) && os.Getenv("TERM") != "dumb"
	total := int64(release.AssetByteSize)
	if total == 0 {
		total = -1
	}
	return &updateReader{ReadCloser: reader, progress: utils.NewUpdateProgress(s.out, total, interactive, cliHeading)}, nil
}

type updateReader struct {
	io.ReadCloser
	progress *utils.UpdateProgress
	complete bool
}

func (r *updateReader) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	r.progress.Write(p[:n])
	r.complete = err == io.EOF
	return n, err
}

func (r *updateReader) Close() error {
	err := r.ReadCloser.Close()
	r.progress.Finish(r.complete && err == nil)
	return err
}
