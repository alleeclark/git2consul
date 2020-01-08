package git

import (
	"encoding/hex"
	"io/ioutil"
	"strings"
	"time"

	git2go "github.com/alleeclark/git2go"
	log "github.com/sirupsen/logrus"
)

type options struct {
	branch                                                           string
	pullDirectory                                                    string
	url                                                              string
	username                                                         string
	publicKeyPath, privateKeyPath, passphrase, gitRSAFingerprintPath string
	fingerPrint                                                      []byte
}

//GitOptions to simplify function signuratures
type GitOptions func(*options) error

//Branch sets branch for git repo
func Branch(branch string) GitOptions {
	return func(o *options) error {
		o.branch = branch
		return nil
	}
}

//PullDir sets directory to pull from
func PullDir(path string) GitOptions {
	return func(o *options) error {
		o.pullDirectory = path
		return nil
	}
}

//URL sets the stash URL
func URL(url string) GitOptions {
	return func(o *options) error {
		o.url = url
		return nil
	}
}

//Username sets the username for stash repo
func Username(username string) GitOptions {
	return func(o *options) error {
		o.username = username
		return nil
	}
}

//PublicKeyPath sets publickey for stash repo
func PublicKeyPath(path string) GitOptions {
	return func(o *options) error {
		o.publicKeyPath = path
		return nil
	}
}

//PrivateKeyPath sets private key for stash repo
func PrivateKeyPath(path string) GitOptions {
	return func(o *options) error {
		o.privateKeyPath = path
		return nil
	}
}

func PassphrasePath(path string) GitOptions {
	return func(o *options) error {
		passphrase, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		o.passphrase = string(passphrase)
		return nil
	}
}

func GitRSAFingerPrintPath(path string) GitOptions {
	return func(o *options) error {
		fingerPrint, err := ioutil.ReadFile(o.gitRSAFingerprintPath)
		// need to add and test other edge cases
		if err != nil {
			return err
		}
		gitFingerPrint := []byte{}
		for i, val := range strings.Split(string(fingerPrint), ":") {
			d, err := hex.DecodeString(val)
			if err != nil {
				return err
			}
			gitFingerPrint[i] = d[0]
		}
		o.fingerPrint = gitFingerPrint
		return nil
	}
}

//FilterFunc as a template for implementing filter based functions
type FilterFunc func(*Collection) bool

//Collection represents the structure needed for a git repository
type Collection struct {
	Commits []*git2go.Commit
	Ref     *git2go.Reference
	*git2go.Repository
}

//Filter function implements the function to be filtered if true
func (c *Collection) Filter(fn FilterFunc) *Collection {
	if fn(c) {
		return c
	}
	return c
}

//ByCommitID filters by a given commit ID
func ByCommitID(id *git2go.Oid) FilterFunc {
	return func(c *Collection) bool {
		commit, err := c.LookupCommit(id)
		if err != nil {
			log.Warningf("Error finding commit id: %s %v", id, err)
			return false
		}
		c.Commits = append(c.Commits, commit)
		return true
	}
}

//ByBranch filters by a given branch
func ByBranch(name string) FilterFunc {
	return func(c *Collection) bool {
		branch, err := c.Repository.LookupBranch(name, git2go.BranchRemote)
		if err != nil {
			log.Warningf("Error finding branch %s %v", name, err)
			return false
		}
		c.Ref = branch.Reference
		return true
	}
}

//ByDate filters by a given date and returns commits in a given date
func ByDate(date time.Time) FilterFunc {
	return func(c *Collection) bool {
		if c.Ref == nil {
			log.Warningln("Error finding ref of the current git collection. Make sure the branch is pushed to origin")
			return false
		}
		revWalk, err := c.Repository.Walk()
		if err != nil {
			log.Warningf("Could not walk repo %v", err)
			return false
		}
		if err := revWalk.PushGlob("*"); err != nil {
			log.Warningf("Error pushing glob %v", err)
			if err := revWalk.Push(c.Ref.Target()); err != nil {
				log.Warningf("Error pushing git reference target %v", err)
				return false
			}
			revWalk.Sorting(git2go.SortTime)
			revWalk.SimplifyFirstParent()
			id := &(git2go.Oid{})
			for revWalk.Next(id) == nil {
				g, err := c.Repository.LookupCommit(id)
				if err != nil {
					log.Warningf("Error finding commit id %v %v", id, err)
				}
				if g.Author().When.Before(date) {
					break
				}
				c.Commits = append(c.Commits, g)
			}
		}
		return true
	}
}
