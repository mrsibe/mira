package updates

import (
	"testing"
	"time"
)

type fakeProvider struct {
	enabled            bool
	automaticChecks    bool
	automaticDownloads bool
	lastCheck          time.Time
	availableVersion   string
	checked            int
}

func (p *fakeProvider) Enabled() bool              { return p.enabled }
func (p *fakeProvider) AutomaticChecks() bool      { return p.automaticChecks }
func (p *fakeProvider) AutomaticDownloads() bool   { return p.automaticDownloads }
func (p *fakeProvider) LastCheck() time.Time       { return p.lastCheck }
func (p *fakeProvider) AvailableVersion() string   { return p.availableVersion }
func (p *fakeProvider) SetAutomaticChecks(on bool) { p.automaticChecks = on }
func (p *fakeProvider) CheckForUpdates()           { p.checked++ }

func TestStatusFormatsVersionTimeAndPreferences(t *testing.T) {
	last := time.Date(2026, 11, 10, 8, 30, 0, 0, time.FixedZone("CST", 8*60*60))
	provider := &fakeProvider{
		enabled:            true,
		automaticChecks:    true,
		automaticDownloads: false,
		lastCheck:          last,
		availableVersion:   "0.3.0",
	}
	service := newService(provider, func() string { return "0.2.0" })

	got := service.GetStatus()
	if got.Version != "0.2.0" || !got.Enabled || !got.AutomaticChecks ||
		got.AutomaticDownloads || got.AvailableVersion != "0.3.0" {
		t.Fatalf("status = %+v", got)
	}
	if got.LastCheck != last.Format(time.RFC3339) {
		t.Fatalf("lastCheck = %q", got.LastCheck)
	}
}

func TestStatusOmitsZeroLastCheck(t *testing.T) {
	service := newService(&fakeProvider{}, func() string { return "0.2.0" })
	if got := service.GetStatus().LastCheck; got != "" {
		t.Fatalf("lastCheck = %q", got)
	}
}

func TestSetAutomaticChecksReturnsUpdatedStatus(t *testing.T) {
	provider := &fakeProvider{}
	service := newService(provider, func() string { return "0.2.0" })

	got := service.SetAutomaticChecks(true)
	if !provider.automaticChecks || !got.AutomaticChecks {
		t.Fatalf("automatic checks were not enabled: %+v", got)
	}
}

func TestCheckForUpdatesDelegates(t *testing.T) {
	provider := &fakeProvider{}
	service := newService(provider, func() string { return "0.2.0" })

	service.CheckForUpdates()
	if provider.checked != 1 {
		t.Fatalf("checks = %d", provider.checked)
	}
}
