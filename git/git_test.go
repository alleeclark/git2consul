package git

import (
	"os"
	"testing"
)

const (
	testDir  = "/var/git2consul/test/"
	sshPath  = "/var/git2consul/config/"
	testRepo = "https://github.com/alleeclark/test-git2consul.git"
)

func TestNewRepository(t *testing.T) {
	collection := NewRepository(
		URL(testRepo),
		PullDir(testDir),
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

func TestOpenRepository(t *testing.T) {
	repo := Open(testDir)
	if repo.Repository == nil {
		t.Log("Repo does not contain any attributes")
		t.Fail()
	}
}

func TestFetch(t *testing.T) {
	opt := options{
		branch:        "origin",
		pullDirectory: testDir,
		url:           testRepo,
		username:      "sre",
		fingerPrint:   []byte(""),
	}
	cloneOpts := CloneOptions(opt.username, opt.password opt.fingerPrint)
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
