package main

import "testing"

func TestGithubRawURLFromContextURLGit(t *testing.T) {
	rawURL, err := githubRawURLFromContextURL(
		"https://github.com/cryptowizard0/vmdocker_agent.git#dac24bbfe8eda6f9c9548a0de6794faec05ffcb2",
		"Dockerfile.sandbox",
	)
	if err != nil {
		t.Fatalf("githubRawURLFromContextURL returned error: %v", err)
	}

	want := "https://raw.githubusercontent.com/cryptowizard0/vmdocker_agent/dac24bbfe8eda6f9c9548a0de6794faec05ffcb2/Dockerfile.sandbox"
	if rawURL != want {
		t.Fatalf("unexpected raw url:\nwant: %s\ngot:  %s", want, rawURL)
	}
}

func TestGithubRawURLFromContextURLArchive(t *testing.T) {
	rawURL, err := githubRawURLFromContextURL(
		"https://github.com/cryptowizard0/vmdocker_agent/archive/refs/heads/feature/flow_optimization.tar.gz",
		"docker/Dockerfile",
	)
	if err != nil {
		t.Fatalf("githubRawURLFromContextURL returned error: %v", err)
	}

	want := "https://raw.githubusercontent.com/cryptowizard0/vmdocker_agent/feature/flow_optimization/docker/Dockerfile"
	if rawURL != want {
		t.Fatalf("unexpected raw url:\nwant: %s\ngot:  %s", want, rawURL)
	}
}

func TestGithubRawURLFromContextURLRejectsUnsupportedHost(t *testing.T) {
	_, err := githubRawURLFromContextURL(
		"https://gitlab.com/example/repo.git#main",
		"Dockerfile",
	)
	if err == nil {
		t.Fatal("expected error for unsupported host")
	}
}
