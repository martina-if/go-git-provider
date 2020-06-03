package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"

	"github.com/google/go-github/v31/github"
	"golang.org/x/oauth2"

	"github.com/weaveworks/go-git-provider/pkg/providers"
)

const (
	EnvVarGitHubToken = "GITHUB_TOKEN"
)

var (
	sshFull = regexp.MustCompile(`ssh://git@github.com/([^/]+)/([^.]+).git`)
	sshShort = regexp.MustCompile(`git@github.com:([^/]+)/([^.]+).git`)

	patterns = []*regexp.Regexp{
		sshFull,
		sshShort,
	}
)

// GitHubProvider accesses the Github API
type GitHubProvider struct {
	owner, repo string
	readOnly    bool
	githubToken string
}

func NewGitHubProvider(repoURL string, readOnly bool) (*GitHubProvider, error) {
	githubToken := os.Getenv(EnvVarGitHubToken)
	if githubToken == "" {
		return nil,fmt.Errorf("%s is not set. Cannot authenticate to github.com", EnvVarGitHubToken)
	}

	repo, err := repoName(repoURL)
	if err != nil {
		return nil, err
	}

	owner, err := repoOwner(repoURL)
	if err != nil {
		return nil, err
	}
	return &GitHubProvider{
			githubToken: githubToken,
			readOnly:    readOnly,
			owner:       owner,
			repo:        repo,
	}, nil
}

func (p *GitHubProvider) AuthorizeSSHKey(ctx context.Context, key providers.SSHKey) error {
	gh := p.getGitHubAPIClient(ctx)

	_, resp, err := gh.Repositories.CreateKey(ctx, p.owner, p.repo, &github.Key{
		Key:      &key.Key,
		Title:    &key.Title,
		ReadOnly: &p.readOnly,
	})

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unable to authorize SSH Key %q. Got StatusCode %s", key.Title, resp.Status)
	}

	return nil
}

func (p *GitHubProvider) Delete(ctx context.Context, title string) error {
	gh := p.getGitHubAPIClient(ctx)

	keys, _, err := gh.Repositories.ListKeys(ctx, p.owner, p.repo, &github.ListOptions{})
	if err != nil {
		return err
	}

	var keyID int64

	for _, key := range keys {
		if key.GetTitle() == title {
			keyID = key.GetID()

			break
		}
	}

	if keyID == 0 {
		return nil
	}

	if _, err := gh.Repositories.DeleteKey(ctx, p.owner, p.repo, keyID); err != nil {
		return err
	}

	return nil
}

func (p *GitHubProvider) getGitHubAPIClient(ctx context.Context) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: p.githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	gh := github.NewClient(tc)

	return gh
}

func repoOwner(repoURL string) (string, error) {
	return findRepoGroup(repoURL, 1)
}

func repoName(repoURL string) (string, error) {
	return findRepoGroup(repoURL, 2)
}

func findRepoGroup(repoURL string, groupNum int) (string, error) {
	if repoURL == "" {
		return "", errors.New("unable to parse empty repo URL")
	}

	for _, p := range patterns {
		m := p.FindStringSubmatch(repoURL)
		if len(m) == 3 {
			return m[groupNum], nil
		}
	}

	return "", fmt.Errorf("unable to parse repo URL %q", repoURL)
}