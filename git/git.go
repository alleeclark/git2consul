package git

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	git2go "github.com/libgit2/git2go/v29"
)

//Fetch remote for a given branch
func (c *Collection) Fetch(opts *git2go.CloneOptions, remoteName string) *Collection {
	if _, err := os.Stat(c.Repository.Path()); os.IsNotExist(err) {
		logrus.WithField("branch", remoteName).Warning("unable to find path of the local repository of branch to clone")
		return nil
	}
	remote, err := c.Repository.Remotes.Lookup(remoteName)
	if err != nil {
		logrus.WithField("error", err).Warning("failed looking up remote repository")
		return c
	}
	var refspecs []string
	if err = remote.Fetch(refspecs, opts.FetchOptions, ""); err != nil {
		logrus.WithField("error", err).Warning("failed fetching remote repository")
		return c
	}
	return c
}

//Open repository
func Open(repoPath string) *Collection {
	repo, err := git2go.OpenRepository(repoPath)
	if err != nil {
		logrus.WithField("error", err).Warning("failed opening repository")
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
			logrus.WithField("error", err).Warning("failed setting option")
			return nil
		}
	}
	_, err := os.Stat(opts.pullDirectory)
	if os.IsExist(err) {
		if repo := Open(opts.pullDirectory); repo == nil {
			cloneOpts := CloneOptions(opts.username, opts.password, opts.publicKeyPath, opts.privateKeyPath, opts.passphrase, opts.fingerPrint)
			if cloneOpts == nil {
				logrus.Warningln("clone options do not exist")
			}
			repopository, err := git2go.Clone(opts.url, opts.pullDirectory, cloneOpts)
			if err != nil && strings.Contains(err.Error(), "exists and is not an empty directory") {
				logrus.Debug("repository already found, so opening it")
				return Open(opts.pullDirectory)
			} else if err != nil {
				logrus.WithFields(logrus.Fields{
					"url":   opts.url,
					"error": err,
				}).Warning("failed cloning url after finding directory")
				return nil
			}
			return &Collection{
				Repository: repopository,
			}
		}
	}
	logrus.WithFields(logrus.Fields{
		"directory": opts.pullDirectory,
	}).Debug("did not find an existing repo so creating the directory")
	if mkirErr := os.MkdirAll(opts.pullDirectory, 0777); mkirErr != nil {
		logrus.WithField("error", err).Debug("failed creating the directory")
	}
	cloneOpts := CloneOptions(opts.username, opts.password, opts.publicKeyPath, opts.privateKeyPath, opts.passphrase, opts.fingerPrint)
	if cloneOpts == nil {
		logrus.Warningln("clone options do not exist")
		return nil
	}
	repoCollection, err := git2go.Clone(opts.url, opts.pullDirectory, cloneOpts)
	if err != nil && strings.Contains(err.Error(), "exists and is not an empty directory") {
		return Open(opts.pullDirectory)
	} else if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
			"url":   opts.url,
		}).Warning("failed cloning url")
		return nil
	}
	return &Collection{
		Repository: repoCollection,
	}
}

//WithIgnoredFiles type to make an optional parameter
type WithIgnoredFiles map[string][]byte

//ListFileChanges returns a map of files that have changed based on filtered commmits found along with the contents
func (c *Collection) ListFileChanges(pullDir string, ignoreFiles ...WithIgnoredFiles) map[string][]byte {
	if len(c.Commits) == 0 {
		logrus.Infof("no commits found to sync contents %d \n", len(c.Commits))
		return nil
	}
	if len(c.Commits) == 1 {
		logrus.Warningf("not enough deltas in the tree to continue")
		return nil
	}
	oldTree, err := c.Commits[0].Tree()
	if err != nil {
		return nil
	}
	newTree, err := c.Commits[len(c.Commits)-1].Tree()
	if err != nil {
		return nil
	}
	diffOptions, err := git2go.DefaultDiffOptions()
	if err != nil {
		logrus.WithField("error", err).Debug("failed getting diff options")
		return nil
	}

	diff, err := c.DiffTreeToTree(oldTree, newTree, &diffOptions)
	if err != nil {
		logrus.WithField("error", err).Warning("failed diffing tree")
		return nil
	}

	numOfDeltas, err := diff.NumDeltas()
	if err != nil {
		logrus.Warningf("failed getting num of deltas %v", err)
		return nil
	}
	if numOfDeltas == 0 {
		logrus.Infoln("no deltas found")
		return nil
	}
	if pullDir[len(pullDir)-1:] == "/" {
		pullDir = pullDir[:len(pullDir)-1]
	}

	fileChanges := make(map[string][]byte, numOfDeltas)
	for delta := 0; delta < numOfDeltas; delta++ {
		diffDelta, err := diff.GetDelta(delta)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"delta": delta,
				"error": err,
			}).Warning("failed getting diff")
		}
		if len(ignoreFiles) > 0 {
			for i := range ignoreFiles {
				if _, ok := ignoreFiles[i][diffDelta.NewFile.Path]; !ok {
					continue
				}
				contents, err := ioutil.ReadFile(pullDir + "/" + diffDelta.NewFile.Path)
				if err != nil || os.IsNotExist(err) {
					logrus.WithFields(logrus.Fields{
						"error": err,
						"path":  diffDelta.NewFile.Path,
					}).Warning("did not map contents %s because it does not exist")
					fileChanges[diffDelta.NewFile.Path] = nil
				}
				fileChanges[diffDelta.NewFile.Path] = contents
			}

		}
		contents, err := ioutil.ReadFile(pullDir + "/" + diffDelta.NewFile.Path)
		if err != nil || os.IsNotExist(err) {
			logrus.WithFields(logrus.Fields{
				"error": err,
				"path":  diffDelta.NewFile.Path,
			}).Warning("did not map contents %s because it does not exist")
			fileChanges[diffDelta.NewFile.Path] = nil
		}
		fileChanges[diffDelta.NewFile.Path] = contents
	}
	return fileChanges
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
	cloneOptions.CheckoutOpts.Strategy = 1
	cloneOptions.FetchOptions.RemoteCallbacks = cbs
	return cloneOptions
}
