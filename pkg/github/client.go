package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// NotFoundError represents a resource not found condition.
// Used by the mock client and checked by IsNotFound.
type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string { return e.Message }

// IsNotFound returns true if the error represents a GitHub 404 Not Found response.
// It checks for both the real go-github ErrorResponse and the mock NotFoundError.
func IsNotFound(err error) bool {
	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) {
		return ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusNotFound
	}
	var nfe *NotFoundError
	return errors.As(err, &nfe)
}

// ClientInterface defines the GitHub client contract
type ClientInterface interface {
	GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error)
	GetRelease(ctx context.Context, owner, repo, tag string) (*github.RepositoryRelease, error)
	ListReleases(ctx context.Context, owner, repo string) ([]*github.RepositoryRelease, error)
	CreateRelease(ctx context.Context, owner, repo string, release *github.RepositoryRelease) (*github.RepositoryRelease, error)
	UploadReleaseAsset(ctx context.Context, owner, repo string, releaseID int64, assetPath, contentType string) (*github.ReleaseAsset, error)
	GetAuthenticatedUser(ctx context.Context) (*github.User, error)
	ForkRepository(ctx context.Context, owner, repo string) (*github.Repository, error)
	CreatePullRequest(ctx context.Context, owner, repo string, pr *github.NewPullRequest) (*github.PullRequest, error)
	GetFileContents(ctx context.Context, owner, repo, path string) (*github.RepositoryContent, error)
	CreateFile(ctx context.Context, owner, repo, path, message string, content []byte) error
	UpdateFile(ctx context.Context, owner, repo, path, message string, content []byte, sha string) error
}

// Ensure Client implements ClientInterface
var _ ClientInterface = (*Client)(nil)

// Client wraps the GitHub client with convenience methods
type Client struct {
	client *github.Client
}

// NewClient creates a new GitHub client with the provided token for authentication.
// If token is empty, an error is returned since GitHub operations require authentication.
func NewClient(token string) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}

	// Configure HTTP client with timeouts to prevent hanging
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), ts)
	// oauth2.NewClient returns a client without timeout, so we set it explicitly
	httpClient.Timeout = 5 * time.Minute

	return &Client{
		client: github.NewClient(httpClient),
	}, nil
}

// GetGitHubToken retrieves GitHub token from environment
func GetGitHubToken() string {
	return os.Getenv("GITHUB_TOKEN")
}

// GetRepository fetches repository information
func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error) {
	repoResp, _, err := c.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository %s/%s: %w", owner, repo, err)
	}
	return repoResp, nil
}

// GetRelease fetches a specific release
func (c *Client) GetRelease(ctx context.Context, owner, repo, tag string) (*github.RepositoryRelease, error) {
	release, _, err := c.client.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to get release %s/%s@%s: %w", owner, repo, tag, err)
	}
	return release, nil
}

// ListReleases fetches all releases for a repository
func (c *Client) ListReleases(ctx context.Context, owner, repo string) ([]*github.RepositoryRelease, error) {
	opt := &github.ListOptions{PerPage: 100}
	var allReleases []*github.RepositoryRelease

	for {
		releases, resp, err := c.client.Repositories.ListReleases(ctx, owner, repo, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to list releases %s/%s: %w", owner, repo, err)
		}

		allReleases = append(allReleases, releases...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allReleases, nil
}

// CreateRelease creates a new release
func (c *Client) CreateRelease(ctx context.Context, owner, repo string, release *github.RepositoryRelease) (*github.RepositoryRelease, error) {
	newRelease, _, err := c.client.Repositories.CreateRelease(ctx, owner, repo, release)
	if err != nil {
		return nil, fmt.Errorf("failed to create release %s/%s: %w", owner, repo, err)
	}
	return newRelease, nil
}

// UploadReleaseAsset uploads an asset to a release
func (c *Client) UploadReleaseAsset(ctx context.Context, owner, repo string, releaseID int64, assetPath, contentType string) (*github.ReleaseAsset, error) {
	// Validate asset path to prevent directory traversal attacks
	absPath, err := filepath.Abs(assetPath)
	if err != nil {
		return nil, fmt.Errorf("invalid asset path: %w", err)
	}

	cleanPath := filepath.Clean(absPath)
	if cleanPath != absPath {
		return nil, fmt.Errorf("invalid asset path: path traversal detected")
	}

	// Verify file exists and is a regular file (not a symlink)
	info, err := os.Lstat(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access asset file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("asset file cannot be a symbolic link")
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("asset path is not a regular file")
	}

	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open asset file: %w", err)
	}
	defer func() { _ = file.Close() }()

	openedInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat opened asset file: %w", err)
	}
	if openedInfo.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("opened asset file cannot be a symbolic link")
	}

	uploadOpts := &github.UploadOptions{
		Name: filepath.Base(assetPath),
	}

	asset, _, err := c.client.Repositories.UploadReleaseAsset(ctx, owner, repo, releaseID, uploadOpts, file)
	if err != nil {
		return nil, fmt.Errorf("failed to upload asset to release %d: %w", releaseID, err)
	}

	return asset, nil
}

// GetAuthenticatedUser returns the authenticated GitHub user
func (c *Client) GetAuthenticatedUser(ctx context.Context) (*github.User, error) {
	user, _, err := c.client.Users.Get(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get authenticated user: %w", err)
	}
	return user, nil
}

// ForkRepository creates a fork of a repository
func (c *Client) ForkRepository(ctx context.Context, owner, repo string) (*github.Repository, error) {
	fork, _, err := c.client.Repositories.CreateFork(ctx, owner, repo, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fork repository %s/%s: %w", owner, repo, err)
	}
	return fork, nil
}

// CreatePullRequest creates a pull request
func (c *Client) CreatePullRequest(ctx context.Context, owner, repo string, pr *github.NewPullRequest) (*github.PullRequest, error) {
	newPR, _, err := c.client.PullRequests.Create(ctx, owner, repo, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request in %s/%s: %w", owner, repo, err)
	}
	return newPR, nil
}

// GetFileContents retrieves the contents of a file in a repository
func (c *Client) GetFileContents(ctx context.Context, owner, repo, path string) (*github.RepositoryContent, error) {
	content, _, _, err := c.client.Repositories.GetContents(ctx, owner, repo, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get contents of %s in %s/%s: %w", path, owner, repo, err)
	}
	return content, nil
}

// CreateFile creates a new file in a repository via the Contents API
func (c *Client) CreateFile(ctx context.Context, owner, repo, path, message string, content []byte) error {
	opts := &github.RepositoryContentFileOptions{
		Message: &message,
		Content: content,
	}
	_, _, err := c.client.Repositories.CreateFile(ctx, owner, repo, path, opts)
	if err != nil {
		return fmt.Errorf("failed to create file %s in %s/%s: %w", path, owner, repo, err)
	}
	return nil
}

// UpdateFile updates an existing file in a repository via the Contents API.
// The sha parameter is the blob SHA of the file being replaced.
func (c *Client) UpdateFile(ctx context.Context, owner, repo, path, message string, content []byte, sha string) error {
	opts := &github.RepositoryContentFileOptions{
		Message: &message,
		Content: content,
		SHA:     &sha,
	}
	_, _, err := c.client.Repositories.UpdateFile(ctx, owner, repo, path, opts)
	if err != nil {
		return fmt.Errorf("failed to update file %s in %s/%s: %w", path, owner, repo, err)
	}
	return nil
}
