package git

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	git2go "github.com/libgit2/git2go/v29"
	"github.com/sirupsen/logrus"
)

//Pull a given remote for a given branch
func (c *Collection) Pull(opts *git2go.CloneOptions, remoteName, branch string) *Collection {
	if _, err := os.Stat(c.Repository.Path()); os.IsNotExist(err) {
		logrus.WithField("branch", remoteName).Warning("unable to find path of the local repository of branch to clone")
		return nil
	}
	remote, err := c.Repository.Remotes.Lookup(remoteName)
	if err != nil {
		logrus.WithError(err).Error("failed looking up remote repository")
		return c
	}
	localBranchRef := fmt.Sprintf("refs/heads/%s", branch)
	if err = remote.Fetch([]string{localBranchRef}, opts.FetchOptions, ""); err != nil {
		logrus.WithError(err).Error("failed fetching remote repository")
		return c
	}
	rawRemoteBranchRef := fmt.Sprintf("refs/remotes/origin/%s", branch)
	remoteBranch, err := c.Repository.References.Lookup(rawRemoteBranchRef)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error":      err,
			"remoteName": remoteName,
		})
		return c
	}
	_, err = c.Repository.References.Lookup(localBranchRef)
	if err != nil {
		_, err := c.Repository.References.Create(localBranchRef, remoteBranch.Target(), true, "")
		if err != nil {
			logrus.WithError(err).Error("failed creating local branch")
			return c
		}
	}
	// need to add logging statements
	if err = c.Repository.SetHead(localBranchRef); err != nil {
		logrus.WithError(err).Error("failed to set head of local branch")
		return c
	}
	if err = c.Repository.CheckoutHead(opts.CheckoutOpts); err != nil {
		logrus.WithError(err).Error("failed to checkout head")
		return c
	}
	head, err := c.Repository.Head()
	if err != nil {
		logrus.WithError(err).Error("failed to get repo's head")
		return c
	}

	annotatedCommit, err := c.Repository.AnnotatedCommitFromRef(remoteBranch)
	if err != nil {
		logrus.WithError(err).Error("failed getting annontated commit from remote branch")
		return c
	}

	mergeHeads := []*git2go.AnnotatedCommit{annotatedCommit}
	analysis, _, err := c.Repository.MergeAnalysis(mergeHeads)
	if err != nil {
		logrus.WithError(err).Error("failed peforming merge analysis")
	}

	switch {
	case analysis&git2go.MergeAnalysisFastForward != 0, analysis&git2go.MergeAnalysisNormal != 0:
		mergeOpts, _ := git2go.DefaultMergeOptions()
		mergeOpts.FileFavor = git2go.MergeFileFavorTheirs
		if err := c.Repository.Merge(mergeHeads, &mergeOpts, opts.CheckoutOpts); err != nil {
			logrus.WithError(err).Error("failed to merge")
			return c
		}
		if _, err = head.SetTarget(remoteBranch.Target(), ""); err != nil {
			logrus.WithError(err).Error("failed updating refs on heads (local) from remotes")
			return c
		}
	}
	logrus.WithField("analysis", analysis).Debug("pull request analysis")
	defer head.Free()
	defer c.Repository.StateCleanup()

	return c
}

//Open repository
func Open(repoPath string) *Collection {
	repo, err := git2go.OpenRepository(repoPath)
	if err != nil {
		logrus.WithError(err).Warning("failed opening repository")
		return nil
	}
	return &Collection{nil, nil, repo}

}

//NewRepository initializes the git repository
func NewRepository(opt ...GitOptions) *Collection {
	opts := defaultCloneOptions
	for _, f := range opt {
		err := f(&opts)
		if err != nil {
			logrus.WithError(err).Error("failed setting option")
			return nil
		}
	}
	_, err := os.Stat(opts.pullDirectory)
	if err != nil {
		if os.IsNotExist(err) {
			cloneOpts := CloneOptions(opts.username, opts.password, opts.publicKeyPath, opts.privateKeyPath, opts.passphrase, opts.fingerPrint)
			if cloneOpts == nil {
				logrus.Warningln("clone options do not exist")
			}
			repository, err := git2go.Clone(opts.url, opts.pullDirectory, cloneOpts)
			if err != nil {
				logrus.WithError(err).Error("failed to clone repo")
				return nil
			}
			logrus.Debug("returning a cloned repo")
			return &Collection{
				Repository: repository,
			}
		}
	}
	logrus.WithFields(logrus.Fields{
		"directory": opts.pullDirectory,
	}).Info("found an existing repo")

	return Open(opts.pullDirectory)
}

//WithIgnoredFiles type to make an optional parameter
type WithIgnoredFiles map[string][]byte

func (c *Collection) ReadFile(gitDir, filePath string) []byte {
	data, err := ioutil.ReadFile(filepath.Join(gitDir, filePath))
	if err != nil {
		logrus.WithError(err).Error("failed reading file")
		return nil
	}
	return data
}

func (c *Collection) GetHeadCommit() *git2go.Commit {
	ref, err := c.Head()
	if err != nil {
		logrus.WithError(err).Error("failed getting head of repo")
		return nil
	}
	return c.getCommit(ref.Target().String())
}

var defaultCloneOptions = options{
	username:       "git2consul",
	password:       "",
	publicKeyPath:  "/var/git2consul/id_rsa.pub",
	privateKeyPath: "/var/git2consul/id_rsa",
	passphrase:     "",
	fingerPrint:    []byte{},
}

//CloneOptions sets all needed options for git fetch, cloning and checkouts
//TODOO make more flexible for different credential types
func CloneOptions(username, password, publickeyPath, privateKeyPath, passphrase string, fingerprint []byte) *git2go.CloneOptions {
	credentialsCallback := func(url string, username string, allowedTypes git2go.CredType) (*git2go.Cred, error) {
		if password == "" {
			logrus.WithField("public-key-path", publickeyPath).Debugf("using sshkeys for auth")
			return git2go.NewCredSshKey(username, publickeyPath, privateKeyPath, passphrase)
		} else if password != "" {
			logrus.Debugln("using user password aut")
			return git2go.NewCredUserpassPlaintext(username, password)
		} else {
			logrus.Debugln("using default git credentials")
			return git2go.NewCredDefault()
		}
	}
	var cbs git2go.RemoteCallbacks
	if len(fingerprint) < 1 {
		certificateCheckCallback := func(cert *git2go.Certificate, valid bool, hostname string) git2go.ErrorCode {
			for i := 0; i < len(fingerprint); i++ {
				if cert.Hostkey.HashMD5[i] != fingerprint[i] {
					logrus.Warningln("remote certificate invalid")
					return git2go.ErrUser
				}
			}
			return 0
		}
		cbs = git2go.RemoteCallbacks{
			CredentialsCallback:      credentialsCallback,
			CertificateCheckCallback: certificateCheckCallback,
		}

	}

	cloneOptions := &git2go.CloneOptions{}
	cloneOptions.FetchOptions = &git2go.FetchOptions{}
	cloneOptions.CheckoutOpts = &git2go.CheckoutOpts{}
	cloneOptions.CheckoutOpts.Strategy = git2go.CheckoutForce
	cloneOptions.FetchOptions.RemoteCallbacks = cbs
	return cloneOptions
}
