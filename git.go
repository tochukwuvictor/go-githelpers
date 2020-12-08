package main

import (
	"fmt"
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

func main() {}

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
func (k KeyPath) SetupGitSSHPubKeys() (*gitSSH.PublicKeys, error) {
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
	Repo                  *git.Repository
	SSHKey                *gitSSH.PublicKeys
	SSHURL                string
	InitialTargetRevision string
	TempDir               string
	VCSClient             interface{} // This package only supports GitLab at the moment
	Worktree              *git.Worktree
}

// NewGitRepo returns a GitRepo with the minimum configs required for using the struct
func NewGitRepo(repoDir, repoURL string, sshKey *gitSSH.PublicKeys) GitRepo {
	return GitRepo{
		Dir:    repoDir,
		SSHKey: sshKey,
		SSHURL: repoURL,
	}
}

// Clone uses a given reference name to clone a Git repo
func (gr GitRepo) Clone(ref plumbing.ReferenceName) (*git.Repository, error) {
	// Clones the repository into the given dir, just as a normal git clone does
	repo, err := git.PlainClone(gr.Dir, false, &git.CloneOptions{
		Auth:          gr.SSHKey,
		URL:           gr.SSHURL,
		ReferenceName: ref,
	})

	return repo, err
}

// CommitAll staged all changes on the provided Worktree
func (gr GitRepo) CommitAll(wt *git.Worktree, commitMsg string) (plumbing.Hash, error) {
	hash, err := wt.Commit(commitMsg, &git.CommitOptions{
		All: true,
	})
	return hash, err
}

// Init uses the stored git repo directory info to initialize a new repo
func (gr GitRepo) Init(isBare bool) error {
	repo, err := git.PlainInit(gr.Dir, isBare)
	if err != nil {
		return err
	}
	gr.Repo = repo

	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	gr.Worktree = wt

	return nil
}

// InitAndPushNewRepo does a full init, commit, and push to the main branch
func (gr GitRepo) InitAndPushNewRepo(commitMsg string) error {
	repo, err := git.PlainInit(gr.Dir, false)
	if err != nil {
		return err
	}
	gr.Repo = repo

	_, err = gr.NewBranch("main", false)
	if err != nil {
		return err
	}

	initFiles := []string{".gitignore", "CODEOWNERS"}
	for _, fileName := range initFiles {
		f := fmt.Sprintf("%s/%s", gr.Dir, fileName)
		yes, err := fileExists(f)
		if err != nil {
			return err
		}
		if yes {
			gr.Worktree.Add(f)
		}
	}

	_, err = gr.Worktree.Commit(commitMsg, &git.CommitOptions{All: false})
	if err != nil {
		return err
	}

	err = gr.Push()

	return err
}

// NewBranch creates a new branch on the provided repo
func (gr GitRepo) NewBranch(name string, uniqSuffix bool) (string, error) {
	newBranch := strings.Replace(name, " ", "-", -1)

	if uniqSuffix {
		now := time.Now()
		epochTs := strconv.FormatInt(now.Unix(), 10)
		newBranch = newBranch + "-" + epochTs
	}

	newBranchRefName := plumbing.NewBranchReferenceName(newBranch)

	wt, err := gr.Repo.Worktree()
	if err != nil {
		return newBranch, err
	}
	gr.Worktree = wt

	err = gr.Worktree.Checkout(&git.CheckoutOptions{
		Branch: newBranchRefName,
		Create: true,
		Keep:   true,
	})

	return newBranch, err
}

// Push sends all staged commits to the default remotes of the provided repo
func (gr GitRepo) Push() error {
	err := gr.Repo.Push(&git.PushOptions{
		Auth:       gr.SSHKey,
		RemoteName: defaultRemoteName,
	})
	return err
}

func fileExists(f string) (bool, error) {
	_, err := os.Stat(f)
	if os.IsNotExist(err) {
		return true, nil
	} else if err != nil {
		return false, err
	}
	return false, nil
}
