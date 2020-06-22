package git

import (
	git2go "github.com/libgit2/git2go/v29"
	"github.com/sirupsen/logrus"
)

type DiffDelta struct {
	// NewFile path of the delta
	NewFile string
	// OldFile path of the delta
	OldFile string
	// type of delta
	Status string
}

func (c *Collection) DifftoHead(oid string) []*DiffDelta {
	ref, err := c.Head()
	defer ref.Free()
	if err != nil {
		logrus.WithError(err).Error("failed to get head")
		return nil
	}
	return diffs(c.Repository, c.getCommit(oid), c.getCommit(ref.Branch().Target().String()))
}

func diffs(r *git2go.Repository, commit1, commit2 *git2go.Commit) []*DiffDelta {
	tree1, err := commit1.Tree()
	if err != nil {
		logrus.WithError(err).Error("failed getting tree for commit")
		return nil
	}

	tree2, err := commit2.Tree()
	if err != nil {
		logrus.WithError(err).Error("failed getting tree for commit")
		return nil
	}

	diffOptions, err := git2go.DefaultDiffOptions()
	if err != nil {
		logrus.WithError(err).Error("failed getting diff options")
		return nil
	}

	diff, err := r.DiffTreeToTree(tree1, tree2, &diffOptions)
	if err != nil {
		logrus.WithError(err).Error("failed to diff from tree to tree")
		return nil
	}

	numOfDeltas, err := diff.NumDeltas()
	if err != nil {
		logrus.WithError(err).Error("failed getting number of deltas")
		return nil
	} else if numOfDeltas == 0 {
		logrus.Infoln("no deltas found")
		return nil
	}

	var diffDeltas []*DiffDelta
	for delta := 0; delta < numOfDeltas; delta++ {
		diffDelta, err := diff.GetDelta(delta)
		if err != nil {
			logrus.WithError(err).WithField("delta", delta).Warningln("did not get diff")
		}
		diffDeltas = append(diffDeltas, &DiffDelta{
			NewFile: diffDelta.NewFile.Path,
			OldFile: diffDelta.OldFile.Path,
			Status:  diffDelta.Status.String(),
		})
	}

	return diffDeltas
}

func (c *Collection) getCommit(commitSha string) *git2go.Commit {
	oid, err := git2go.NewOid(commitSha)
	if err != nil {
		logrus.WithError(err).Error("failed getting new oid")
		return nil
	}
	commit, err := c.Repository.LookupCommit(oid)
	if err != nil {

		logrus.WithError(err).Error("failed looking up commit")
		return nil
	}
	return commit
}
