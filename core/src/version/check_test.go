package version

import (
	"errors"
	"testing"
	"time"
)

func TestCheckForUpdateShortCircuits(t *testing.T) {
	originalCache := cache
	originalFetcher := latestReleaseGetter
	cache = nil
	latestReleaseGetter = func() (*githubRelease, error) {
		t.Fatal("fetcher should not be called for disabled or dev versions")
		return nil, nil
	}
	t.Cleanup(func() {
		cache = originalCache
		latestReleaseGetter = originalFetcher
	})

	for _, currentVersion := range []string{"", "development"} {
		info := CheckForUpdate(currentVersion, false)
		if info.UpdateAvailable {
			t.Fatalf("expected no update for version %q", currentVersion)
		}
	}

	info := CheckForUpdate("1.2.3", true)
	if info.UpdateAvailable {
		t.Fatal("expected no update when checks are disabled")
	}
	if info.LatestVersion != "1.2.3" {
		t.Fatalf("expected disabled check to echo current version, got %q", info.LatestVersion)
	}
}

func TestCheckForUpdateUsesFetcherAndSemverComparison(t *testing.T) {
	originalCache := cache
	originalFetcher := latestReleaseGetter
	cache = nil
	t.Cleanup(func() {
		cache = originalCache
		latestReleaseGetter = originalFetcher
	})

	latestReleaseGetter = func() (*githubRelease, error) {
		return &githubRelease{TagName: "v1.3.0"}, nil
	}

	info := CheckForUpdate("v1.2.3", false)
	if !info.UpdateAvailable {
		t.Fatal("expected update to be available")
	}
	if info.LatestVersion != "v1.3.0" {
		t.Fatalf("expected latest version to preserve tag name, got %q", info.LatestVersion)
	}
	if info.ReleaseURL != "https://github.com/clidey/whodb/releases/tag/v1.3.0" {
		t.Fatalf("unexpected release URL: %q", info.ReleaseURL)
	}

	cache = nil
	latestReleaseGetter = func() (*githubRelease, error) {
		return &githubRelease{TagName: "v1.2.0"}, nil
	}

	info = CheckForUpdate("1.2.3", false)
	if info.UpdateAvailable {
		t.Fatalf("did not expect older release to be treated as an update: %#v", info)
	}

	cache = nil
	latestReleaseGetter = func() (*githubRelease, error) {
		return &githubRelease{TagName: "not-a-semver"}, nil
	}
	info = CheckForUpdate("1.2.3", false)
	if info.UpdateAvailable {
		t.Fatalf("did not expect invalid semver to produce update: %#v", info)
	}
}

func TestCheckForUpdateHandlesErrorsAndCacheHits(t *testing.T) {
	originalCache := cache
	originalFetcher := latestReleaseGetter
	cache = nil
	t.Cleanup(func() {
		cache = originalCache
		latestReleaseGetter = originalFetcher
	})

	latestReleaseGetter = func() (*githubRelease, error) {
		return nil, errors.New("network down")
	}
	info := CheckForUpdate("1.2.3", false)
	if info.UpdateAvailable {
		t.Fatalf("expected errors to return no update, got %#v", info)
	}

	cache = &cachedResult{
		info: UpdateInfo{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "v1.1.0",
			UpdateAvailable: true,
			ReleaseURL:      "https://example.com/release",
		},
		checkedAt: time.Now(),
	}
	fetchCalls := 0
	latestReleaseGetter = func() (*githubRelease, error) {
		fetchCalls++
		return &githubRelease{TagName: "v9.9.9"}, nil
	}

	info = CheckForUpdate("1.0.0", false)
	if fetchCalls != 0 {
		t.Fatalf("expected cached result to skip fetcher, got %d calls", fetchCalls)
	}
	if !info.UpdateAvailable || info.LatestVersion != "v1.1.0" {
		t.Fatalf("expected cached info to be returned, got %#v", info)
	}
}
