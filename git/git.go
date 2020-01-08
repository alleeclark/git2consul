package git

import (
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"

	git2go "github.com/alleeclark/git2go"
)

//Fetch remote for a given branch
func (c *Collection) Fetch(opts *git2go.CloneOptions, branchName string) *Collection {
	if _, err := os.Stat(c.Repository.Path()); os.IsNotExist(err) {
		log.Warningf("Unable to find path of the local repository of branch %s to clone", branchName)
		return nil
	}
	remote, err := c.Repository.Remotes.Lookup(branchName)
	if err != nil {
		log.Warningf("Error looking up remote repository %v", err)
		return c
	}
	var refspecs []string
	if err = remote.Fetch(refspecs, opts.FetchOptions, ""); err != nil {
		log.Warningf("Error fetching remote repository %v", err)
		return c
	}
	return c
}

//Open repository
func Open(repoPath string) *Collection {
	repo, err := git2go.OpenRepository(repoPath)
	if err != nil {
		log.Warningf("Error opening repository %v", err)
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
			log.Warningf("Error setting option %v", err)
			return nil
		}
	}
	_, err := os.Stat(opts.pullDirectory)
	if os.IsExist(err) {
		return Open(opts.pullDirectory)
	} else if err != nil {
		log.Warningf("Error finding repo path %s: %v so creating the directory", opts.pullDirectory, err)
		if mkirErr := os.MkdirAll(opts.pullDirectory, 0777); mkirErr != nil {
			log.Warningf("Error creating the directory %v", err)
		}
	}
	cloneOpts := CloneOptions(opts.username, opts.fingerPrint)
	repo, err := git2go.Clone(opts.url, opts.pullDirectory, cloneOpts)
	if err != nil {
		log.Warningf("Error cloning url %s %v", opts.url, err)
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
	if len(c.Commits) < 1 {
		log.Infoln("No commits found to sync to contents")
		return nil
	}
	diffOptions, err := git2go.DefaultDiffOptions()
	if err != nil {
		log.Warningf("Error getting diff options %v", err)
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

	diff, err := c.DiffTreeToTree(oldTree, newTree, &diffOptions)
	if err != nil {
		log.Warningf("Error diffing tree %v", err)
		return nil
	}

	numOfDeltas, err := diff.NumDeltas()
	if err != nil {
		log.Warningf("Error getting num of deltas %v", err)
		return nil
	}
	fileChanges := make(map[string][]byte, numOfDeltas)
	for d := 0; d < numOfDeltas; d++ {
		diffDelta, err := diff.GetDelta(d)
		if err != nil {
			log.Warningf("Error getting diff at %d %v", d, err)
		}
		if len(ignoreFiles) > 0 {
			if _, ok := ignoreFiles[0][diffDelta.NewFile.Path]; !ok {
				contents, err := ioutil.ReadFile(pullDir + "/" + diffDelta.NewFile.Path)
				if err != nil || os.IsNotExist(err) {
					log.Warningf("Did not map contents %s becuase it does not exist %v", diffDelta.NewFile.Path, err)
					fileChanges[diffDelta.NewFile.Path] = nil
				}
				fileChanges[diffDelta.NewFile.Path] = contents
			}
		}
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
					log.Warningln("Remote certificate invalid")
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
