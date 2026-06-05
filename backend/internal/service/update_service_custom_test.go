package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

type customUpdateTestCache struct{}

func (customUpdateTestCache) GetUpdateInfo(context.Context) (string, error) {
	return "", errors.New("cache should not be used for custom builds")
}

func (customUpdateTestCache) SetUpdateInfo(context.Context, string, time.Duration) error {
	return errors.New("cache should not be written for custom builds")
}

type customUpdateTestGitHubClient struct {
	fetchCalled bool
}

func (c *customUpdateTestGitHubClient) FetchLatestRelease(context.Context, string) (*GitHubRelease, error) {
	c.fetchCalled = true
	return nil, errors.New("github should not be queried for custom builds")
}

func (c *customUpdateTestGitHubClient) DownloadFile(context.Context, string, string, int64) error {
	return errors.New("download should not be used for custom builds")
}

func (c *customUpdateTestGitHubClient) FetchChecksumFile(context.Context, string) ([]byte, error) {
	return nil, errors.New("checksum should not be used for custom builds")
}

func TestCustomBuildDisablesOnlineUpdate(t *testing.T) {
	client := &customUpdateTestGitHubClient{}
	svc := NewUpdateService(customUpdateTestCache{}, client, "0.1.117-custom", "release")

	info, err := svc.CheckUpdate(context.Background(), true)
	if err != nil {
		t.Fatalf("CheckUpdate returned error: %v", err)
	}
	if client.fetchCalled {
		t.Fatal("CheckUpdate queried GitHub for a custom build")
	}
	if info.HasUpdate {
		t.Fatal("custom build should not report online updates")
	}
	if info.CurrentVersion != "0.1.117-custom" || info.LatestVersion != "0.1.117-custom" {
		t.Fatalf("unexpected version info: current=%q latest=%q", info.CurrentVersion, info.LatestVersion)
	}
	if info.Warning == "" {
		t.Fatal("custom build update response should explain why online update is disabled")
	}

	if err := svc.PerformUpdate(context.Background()); err == nil {
		t.Fatal("PerformUpdate should reject online update for custom builds")
	}
}
