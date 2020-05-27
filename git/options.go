package git

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	git2go "github.com/libgit2/git2go/v29"
	"github.com/sirupsen/logrus"
)

type options struct {
	branch                                                           string
	pullDirectory                                                    string
	url                                                              string
	username, password                                               string
	publicKeyPath, privateKeyPath, passphrase, gitRSAFingerprintPath string
	fingerPrint                                                      []byte
}

//GitOptions to simplify function signuratures
type GitOptions func(*options) error

//Branch sets branch for the remote repo
func Branch(branch string) GitOptions {
	return func(o *options) error {
		o.branch = fmt.Sprintf("refs/remotes/origin/%s", branch)
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

//URL sets the repo
func URL(url string) GitOptions {
	return func(o *options) error {
		o.url = url
		return nil
	}
}

//Username sets the username for repo
func Username(username string) GitOptions {
	return func(o *options) error {
		o.username = username
		return nil
	}
}

//Password sets the password for repo
func Password(password string) GitOptions {
	return func(o *options) error {
		o.password = password
		return nil
	}
}

//PublicKeyPath sets publickey for repo
func PublicKeyPath(path string) GitOptions {
	return func(o *options) error {
		o.publicKeyPath = path
		return nil
	}
}

//PrivateKeyPath sets private key for repo
func PrivateKeyPath(path string) GitOptions {
	return func(o *options) error {
		o.privateKeyPath = path
		return nil
	}
}

//PassphrasePath to read from
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

//RSAFingerPrintPath to read from
func RSAFingerPrintPath(path string) GitOptions {
	return func(o *options) error {
		fingerPrint, err := ioutil.ReadFile(o.gitRSAFingerprintPath)
		//need to add and test other edge cases
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
	fileChanges []string
	hash        string
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
		if id.IsZero() {
			logrus.Trace("no commit id found")
			return false
		}
		commit, err := c.LookupCommit(id)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"id":    id,
				"error": err,
			}).Warning("error finding commit id")
			return false
		}
		c.Commits = append(c.Commits, commit)
		return true
	}
}

//ByBranch filters by a given branch
func ByBranch(branchName string) FilterFunc {
	return func(c *Collection) bool {
		branch, err := c.Repository.LookupBranch(branchName, git2go.BranchAll)
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{"name": branchName, "option": "ByBranch"}).Error("failed finding branch")
			return false
		}
		c.Ref = branch.Reference
		return true
	}
}

//ByDate filters by a given date and returns commits in a given date
func ByDate(date time.Time) FilterFunc {
	return func(c *Collection) bool {
		if c.Ref == nil || date.IsZero() {
			logrus.Warningln("failed finding ref of the current git collection. Make sure the branch is pushed to origin")
			return false
		}
		revWalk, err := c.Repository.Walk()
		if err != nil {
			logrus.WithError(err).Error("could not walk repo")
			return false
		}
		if err := revWalk.PushGlob("*"); err != nil {
			logrus.WithError(err).Error("failed pushing glob")
			return false
		}
		if err := revWalk.Push(c.Ref.Target()); err != nil {
			logrus.WithError(err).Error("failed pushing git reference target")
			return false
		}
		revWalk.Sorting(git2go.SortTime)
		revWalk.SimplifyFirstParent()
		id := &(git2go.Oid{})
		oldCount := 0
		for revWalk.Next(id) == nil {
			commit, err := c.Repository.LookupCommit(id)
			if err != nil {
				logrus.WithFields(logrus.Fields{"id": id, "error": err}).Warning("failed finding commit")
			}
			if commit.Author().When.UTC().Before(date) {
				if oldCount < 1 {
					c.Commits = append(c.Commits, commit)
					oldCount++
				}
				continue
			}
			c.Commits = append(c.Commits, commit)
		}
		return true
	}
}

//ByTopo sorts by topolgical order from the last commit in the collection
func ByTopo() FilterFunc {
	return func(c *Collection) bool {
		if c.Ref == nil || c.Commits == nil {
			logrus.WithField("commits", len(c.Commits)).Warning("did not find any refs or list of commits")
			return false
		}

		revWalk, err := c.Repository.Walk()
		if err != nil {
			logrus.WithError(err).Warning("failed to walk the repo")
			return false
		}

		if err := revWalk.PushGlob("*"); err != nil {
			logrus.WithError(err).Warning("failed to push glob")
			return false
		}
		if err := revWalk.Push(c.Ref.Target()); err != nil {
			logrus.WithError(err).Warning("failed pushing git reference")
		}
		revWalk.Sorting(git2go.SortTopological)
		id := &(git2go.Oid{})
		for revWalk.Next(id) == nil {
			commit, err := c.Repository.LookupCommit(id)
			if err != nil {
				logrus.WithError(err).WithFields(logrus.Fields{"id": id}).Warning("failed finding commit")
				continue
			}
			if commit.Id().Equal(c.Commits[0].AsObject().Id()) {
				break
			}
			c.Commits = append(c.Commits, commit)
		}
		return true
	}
}

//ByHead gets
func ByHead() FilterFunc {
	return func(c *Collection) bool {
		c.Commits = nil
		obj, err := c.Repository.RevparseSingle(c.hash)
		if err != nil {
			logrus.WithError(err).Error("failed to rev parse single head")
		}
		commit, err := obj.AsCommit()
		if err != nil {
			logrus.WithError(err).Error("error changing object as commit")
			return false
		}
		tree, err := commit.AsTree()
		if err != nil {
			logrus.WithError(err).Error("error changing object as tree")
			return false
		}
		diffOpts, err := git2go.DefaultDiffOptions()
		if err != nil {
			logrus.WithError(err).Error("failed getting default diff options")
		}
		diff, err := c.DiffTreeToWorkdir(tree, &diffOpts)
		if err != nil {
			logrus.WithError(err).Error("failed to diff tree to work dir")
		}
		deltas, err := diff.NumDeltas()
		if err != nil {
			logrus.WithError(err).Error("failed to get num of deltas")
		}
		for i := 0; i < deltas; i++ {
			diffDelta, err := diff.GetDelta(i)
			if err != nil {
				logrus.WithError(err).Error("failed getting detla")
			}
			c.fileChanges = append(c.fileChanges, diffDelta.NewFile.Path)
		}
		return true
	}
}
