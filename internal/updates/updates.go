// Package updates exposes MyGo's signed updater to the settings page.
package updates

import (
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater"
)

// Provider is the updater API used by the settings page. Tests replace it.
type Provider interface {
	Enabled() bool
	AutomaticChecks() bool
	AutomaticDownloads() bool
	LastCheck() time.Time
	AvailableVersion() string
	SetAutomaticChecks(bool)
	CheckForUpdates()
}

type mygoProvider struct{}

func (mygoProvider) Enabled() bool              { return mygo.Updater.Enabled() }
func (mygoProvider) AutomaticChecks() bool      { return updater.AutomaticChecks() }
func (mygoProvider) AutomaticDownloads() bool   { return updater.AutomaticDownloads() }
func (mygoProvider) LastCheck() time.Time       { return updater.LastCheck() }
func (mygoProvider) AvailableVersion() string   { return updater.AvailableVersion() }
func (mygoProvider) SetAutomaticChecks(on bool) { updater.SetAutomaticChecks(on) }
func (mygoProvider) CheckForUpdates()           { updater.CheckForUpdates() }

// UpdateStatus is the frontend-safe state of the update service.
type UpdateStatus struct {
	Version            string `json:"version"`
	Enabled            bool   `json:"enabled"`
	AutomaticChecks    bool   `json:"automaticChecks"`
	AutomaticDownloads bool   `json:"automaticDownloads"`
	AvailableVersion   string `json:"availableVersion,omitempty"`
	LastCheck          string `json:"lastCheck,omitempty"`
}

// Service is bound to the page as `MiraUpdate`.
type Service struct {
	provider Provider
	version  func() string
}

// New creates the bound update service.
func New() *Service {
	return newService(mygoProvider{}, mygo.App.Version)
}

func newService(provider Provider, version func() string) *Service {
	return &Service{provider: provider, version: version}
}

// GetStatus returns the current update preferences and check state.
func (s *Service) GetStatus() UpdateStatus {
	status := UpdateStatus{
		Version:            s.version(),
		Enabled:            s.provider.Enabled(),
		AutomaticChecks:    s.provider.AutomaticChecks(),
		AutomaticDownloads: s.provider.AutomaticDownloads(),
		AvailableVersion:   s.provider.AvailableVersion(),
	}
	if last := s.provider.LastCheck(); !last.IsZero() {
		status.LastCheck = last.Format(time.RFC3339)
	}
	return status
}

// SetAutomaticChecks turns background update checks on or off.
func (s *Service) SetAutomaticChecks(on bool) UpdateStatus {
	s.provider.SetAutomaticChecks(on)
	return s.GetStatus()
}

// CheckForUpdates opens the native update window.
func (s *Service) CheckForUpdates() {
	s.provider.CheckForUpdates()
}
