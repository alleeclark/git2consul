package git

import (
	"os"
	"testing"
)

const (
	testDir = "/var/git2consul/test/"
	sshPath = "/var/git2consul/config/"
)

func TestOpenRepository(t *testing.T) {
	repo := Open(testDir)
	if repo == nil {
		t.Log("Repo does not contain any attributes")
		t.Fail()
	}
}

func TestNewRepository(t *testing.T) {
	collection := NewRepository(
		URL("ssh://git@github.com:alleeclark/git2consul.git"),
		PullDir(testDir),
		Username("sre"),
	)

	if collection.Repository == nil {
		t.Log("Did not get a repository")
		t.Fail()
	}
	_, err := os.Stat(testDir)
	if os.IsNotExist(err) {
		t.Logf("Pull directory does not exist %v", err)
		t.Fail()
	}
}

func TestFetch(t *testing.T) {
	opt := options{
		branch:        "origin",
		pullDirectory: testDir,
		url:           "ssh://git@github.com:alleeclark/git2consul.git",
		username:      "sre",
		fingerPrint:   []byte(""),
	}
	cloneOpts := CloneOptions(opt.username, opt.fingerPrint)
	if cloneOpts == nil {
		t.Log("clone opts returned null")
		t.Fail()
	}
	repo := Open(testDir)
	repo = repo.Fetch(cloneOpts, opt.branch)
	if repo.Repository == nil {
		t.Log("Error Fetching repository")
		t.Fail()
	}
}
