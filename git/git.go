package git

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	git2go "github.com/alleeclark/git2go"
)

//Fetch remote for a given branch
func (c *Collection) Fetch(opts *git2go.CloneOptions, remoteName string) *Collection {
	if _, err := os.Stat(c.Repository.Path()); os.IsNotExist(err) {
		logrus.Warningf("Unable to find path of the local repository of branch %s to clone", remoteName)
		return nil
	}
	remote, err := c.Repository.Remotes.Lookup(remoteName)
	if err != nil {
		logrus.Warningf("Failed looking up remote repository %v", err)
		return c
	}
	var refspecs []string
	if err = remote.Fetch(refspecs, opts.FetchOptions, ""); err != nil {
		logrus.Warningf("Failed fetching remote repository %v", err)
		return c
	}
	return c
}

//Open repository
func Open(repoPath string) *Collection {
	repo, err := git2go.OpenRepository(repoPath)
	if err != nil {
		logrus.Warningf("Failed opening repository %v", err)
		return nil
	}
	return &Collection{nil, nil, repo}

}

//NewRepository initalizes the git repository
func NewRepository(opt ...GitOptions) *Collection {
	opts := defaultCloneOptions
	for _, f := range opt {
		err := f(&opts)
		if err != nil {
			logrus.Warningf("Failed setting option %v", err)
			return nil
		}
	}
	_, err := os.Stat(opts.pullDirectory)
	if os.IsExist(err) {
		return Open(opts.pullDirectory)
	} else if err != nil {
		logrus.Debug("Did not find an existing repo %s: %v so creating the directory", opts.pullDirectory, err)
		if mkirErr := os.MkdirAll(opts.pullDirectory, 0777); mkirErr != nil {
			logrus.Warningf("Failed creating the directory %v", err)
		}
	}
	cloneOpts := CloneOptions(opts.username, opts.fingerPrint)
	repo, err := git2go.Clone(opts.url, opts.pullDirectory, cloneOpts)
	if err != nil && strings.Contains(err.Error(), "exists and is not an empty directory") {
		logrus.Debug("Repository already found, so opening it")
		return Open(opts.pullDirectory)
	} else if err != nil || repo == nil {
		logrus.Warningf("Failed cloning url %s %v", opts.url, err)
		return nil
	}
	return &Collection{
		Repository: repo,
	}
}

//WithIgnoredFiles type to make an optional parameter
type WithIgnoredFiles map[string][]byte

//ListFileChanges returns a map of files that have changed based on filtered commmits found along with the contents
func (c *Collection) ListFileChanges(pullDir string, ignoreFiles ...WithIgnoredFiles) map[string][]byte {
	if len(c.Commits) == 0 {
		logrus.Infoln("No commits found to sync contents %d", len(c.Commits))
		return nil
	}
	if len(c.Commits) == 1 {
		logrus.Warningf("Not enough deltas in the tree to continue")
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
		logrus.Warningf("Failed getting diff options %v", err)
		return nil
	}

	diff, err := c.DiffTreeToTree(oldTree, newTree, &diffOptions)
	if err != nil {
		logrus.Warningf("Failed diffing tree %v", err)
		return nil
	}

	numOfDeltas, err := diff.NumDeltas()
	if err != nil {
		logrus.Warningf("Failed getting num of deltas %v", err)
		return nil
	}
	if numOfDeltas == 0 {
		logrus.Info("No deltas found")
		return nil
	}

	fileChanges := make(map[string][]byte, numOfDeltas)
	for delta := 0; delta < numOfDeltas; delta++ {
		diffDelta, err := diff.GetDelta(delta)
		if err != nil {
			logrus.Warningf("Failed getting diff at %d %v", delta, err)
		}
		if len(ignoreFiles) > 0 {
			if _, ok := ignoreFiles[0][diffDelta.NewFile.Path]; !ok {
				continue
			}
			contents, err := ioutil.ReadFile(pullDir + "/" + diffDelta.NewFile.Path)
			if err != nil || os.IsNotExist(err) {
				logrus.Warningf("Did not map contents %s becuase it does not exist %v", diffDelta.NewFile.Path, err)
				fileChanges[diffDelta.NewFile.Path] = nil
			}
			fileChanges[diffDelta.NewFile.Path] = contents
		}
		contents, err := ioutil.ReadFile(pullDir + "/" + diffDelta.NewFile.Path)
		if err != nil || os.IsNotExist(err) {
			logrus.Warningf("Did not map contents %s becuase it does not exist %v", diffDelta.NewFile.Path, err)
			fileChanges[diffDelta.NewFile.Path] = nil
		}
		fileChanges[diffDelta.NewFile.Path] = contents
	}
	return fileChanges
}

var defaultCloneOptions = options{
	username:       "git2consul",
	publicKeyPath:  "/var/git2consul/id_rsa.pub",
	privateKeyPath: "/var/git2consul/id_rsa",
	passphrase:     "",
	fingerPrint:    []byte{},
}

//CloneOptions sets all needed options for git fetch, cloning and checkouts
//TODOO make more flexible for different credential types
func CloneOptions(username string, gitRSAFingerprint []byte) *git2go.CloneOptions {
	credentialsCallback := func(url string, username string, allowedTypes git2go.CredType) (git2go.ErrorCode, *git2go.Cred) {
		ret, cred := git2go.NewCredSshKeyFromAgent(username)
		return git2go.ErrorCode(ret), &cred
	}
	var cbs git2go.RemoteCallbacks
	if len(gitRSAFingerprint) < 1 {
		certificateCheckCallback := func(cert *git2go.Certificate, valid bool, hostname string) git2go.ErrorCode {
			for i := 0; i < len(gitRSAFingerprint); i++ {
				if cert.Hostkey.HashMD5[i] != gitRSAFingerprint[i] {
					logrus.Warningln("Remote certificate invalid")
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

func withSSH(username, publicKeyPath, privateKeyPath, passharse string) (int, git2go.Cred) {
	if username != "" && publicKeyPath != "" && privateKeyPath != "" && passharse != "" {
		return git2go.NewCredSshKey(username, publicKeyPath, privateKeyPath, passharse)
	}
	return git2go.NewCredSshKeyFromAgent(username)
}
