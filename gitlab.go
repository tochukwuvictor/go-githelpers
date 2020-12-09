package githelpers

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xanzy/go-gitlab"
)

var (
	defaultListOpts = gitlab.ListOptions{PerPage: 1000}
)

// AddGitlabClient takes a Gitlab token and saves the client to the GitRepo receiver
func (gr *GitRepo) AddGitlabClient(vcsToken string) error {
	c, err := gitlab.NewClient(vcsToken)
	gr.VCSClient = c
	return err
}

func (gr *GitRepo) getGitlabGroups() (groups []*gitlab.Group, resp *gitlab.Response, err error) {
	// Move list groups logic into a new func to DRY out the client declaration and
	// allow retrieval of a param other than ID
	client := gr.VCSClient.(*gitlab.Client)
	groups, resp, err = client.Groups.ListGroups(&gitlab.ListGroupsOptions{
		ListOptions: defaultListOpts,
	})
	return groups, resp, err
}

func (gr *GitRepo) getGitlabGroupID(groupPath string) (id int, resp *gitlab.Response, err error) {
	groups, resp, err := gr.getGitlabGroups()
	for _, g := range groups {
		if g.FullPath == groupPath {
			id = g.ID
		}
	}
	return id, resp, err
}

func (gr *GitRepo) getGitlabProjectID(url string) (id int, resp *gitlab.Response, err error) {
	// Move list projects logic into a new func to DRY out the client declaration and
	// allow retrieval of a param other than ID
	client := gr.VCSClient.(*gitlab.Client)
	_, parentGroupPath, name := splitRepoURL(url)

	parentID, _, err := gr.getGitlabGroupID(parentGroupPath)

	projects, resp, err := client.Groups.ListGroupProjects(parentID, &gitlab.ListGroupProjectsOptions{ListOptions: defaultListOpts})

	for _, p := range projects {
		// fmt.Printf("Checking whether %s matches %s\n", p.Path, name)
		if p.Path == name {
			id = p.ID
		}
	}
	return id, resp, err
}

// NewGitlabMergeRequest creates a new MR in Gitlab
func (gr *GitRepo) NewGitlabMergeRequest(commitMsg, src, dest string) (mr *gitlab.MergeRequest, resp *gitlab.Response, err error) {
	c := gr.VCSClient.(*gitlab.Client)

	mrOpts := &gitlab.CreateMergeRequestOptions{
		Title:        &commitMsg,
		SourceBranch: &src,
		TargetBranch: &dest,
	}
	pid, resp, err := gr.getGitlabProjectID(gr.SSHURL)
	if err != nil {
		return mr, resp, err
	}

	mr, resp, err = c.MergeRequests.CreateMergeRequest(pid, mrOpts)
	return mr, resp, err
}

// ShowPwd shows the present working directory
func (gr *GitRepo) ShowPwd() (err error) {
	pwd, err := os.Getwd()
	fmt.Println(pwd)
	return err
}

// ListFiles prints all files in a directory
func (gr *GitRepo) ListFiles(dir string) (err error) {
	files, err := gr.getFiles(dir)
	if err != nil {
		return err
	}
	for _, file := range files {
		fmt.Println(file)
	}
	return err
}

func (gr *GitRepo) getFiles(dir string) (files []string, err error) {
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		files = append(files, path)
		return nil
	})
	return files, err
}
