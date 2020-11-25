package main

import (
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	gitSSH "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh"
)

const (
	defaultRemoteName = "origin"
)

// TempDir holds the directory name of the tmp dir created by NewTempDir().
// It should probably store the fullpath instead
type TempDir struct {
	DirName string
}

// EnterNewTempDir creates a new tmp directory, makes it the working directory,
// and stores it in the returned TempDir struct
func EnterNewTempDir() (t TempDir, err error) {
	dir, err := os.Getwd()
	if err != nil {
		return TempDir{}, err
	}

	d, err := ioutil.TempDir(dir, "")
	if err != nil {
		return TempDir{}, err
	}

	err = os.Chdir(d)
	if err != nil {
		return TempDir{}, err
	}

	return TempDir{DirName: d}, nil
}

// CleanTempDir removes the tmp directory stored in the TempDir struct
func (t *TempDir) CleanTempDir() (err error) {
	return os.RemoveAll(t.DirName)
}

// KeyPath type handles managing the retrieval of SSH public keys
type KeyPath string

// SetupGitSSHPubKeys fetches SSH public keys based on the key path
func (k KeyPath) SetupGitSSHPubKeys() (pubKeys *gitSSH.PublicKeys, err error) {

	pem, err := ioutil.ReadFile(string(k))
	if err != nil {
		return &gitSSH.PublicKeys{}, err
	}

	signer, err := ssh.ParsePrivateKey(pem)
	if err != nil {
		return &gitSSH.PublicKeys{}, err
	}

	return &gitSSH.PublicKeys{
		User:   "git",
		Signer: signer,
		HostKeyCallbackHelper: gitSSH.HostKeyCallbackHelper{
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
	}, nil
}

// GitRepo represents a collection of the git repository name, SSH URL, and the configuration that specifies what file content to change and how
type GitRepo struct {
	CommitMsg             string
	Dir                   string
	Namespace             string
	SSHKey                *gitSSH.PublicKeys
	SSHURL                string
	InitialTargetRevision string
	TempDir               string
	VCSClient             interface{} // This package only supports GitLab at the moment
}

// Clone uses a given reference name to clone a Git repo
func (gr GitRepo) Clone(ref plumbing.ReferenceName) (repo *git.Repository, err error) {
	// Clones the repository into the given dir, just as a normal git clone does
	repo, err = git.PlainClone(gr.Dir, false, &git.CloneOptions{
		Auth:          gr.SSHKey,
		URL:           gr.SSHURL,
		ReferenceName: ref,
	})

	return repo, err
}

// NewBranch creates a new branch on the provided repo
func (gr GitRepo) NewBranch(repo *git.Repository, name string) (newBranch string, wt *git.Worktree, err error) {
	now := time.Now()
	epochTs := strconv.FormatInt(now.Unix(), 10)

	newBranch = strings.Replace(name, " ", "-", -1) + "-" + epochTs
	newBranchRefName := plumbing.NewBranchReferenceName(newBranch)

	wt, err = repo.Worktree()
	err = wt.Checkout(&git.CheckoutOptions{
		Branch: newBranchRefName,
		Create: true,
		Keep:   true,
	})

	return newBranch, wt, err
}

// CommitAll staged all changes on the provided Worktree
func (gr GitRepo) CommitAll(wt *git.Worktree, commitMsg string) (hash plumbing.Hash, err error) {
	hash, err = wt.Commit(commitMsg, &git.CommitOptions{
		All: true,
	})
	return hash, err
}

// Push sends all staged commits to the default remotes of the provided repo
func (gr GitRepo) Push(repo *git.Repository) (err error) {
	repo.Push(&git.PushOptions{
		Auth:       gr.SSHKey,
		RemoteName: defaultRemoteName,
	})
	return err
}
