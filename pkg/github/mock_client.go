package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/github"
)

// Ensure MockClient implements ClientInterface
var _ ClientInterface = (*MockClient)(nil)

// MockClient is a mock implementation of the GitHub client for testing
type MockClient struct {
	Repositories  map[string]*github.Repository
	Releases      map[string][]*github.RepositoryRelease
	Users         map[string]*github.User
	UploadedAssets []string // tracks asset paths passed to UploadReleaseAsset
	FileContents   map[string]*github.RepositoryContent // key: "owner/repo/path"
	CreatedFiles   map[string][]byte                    // key: "owner/repo/path", value: content
	UpdatedFiles   map[string][]byte                    // key: "owner/repo/path", value: content
	ErrorToReturn  error
	UploadError    error // if non-nil, returned by UploadReleaseAsset instead of ErrorToReturn
	ContentsError  error // if non-nil, returned by GetFileContents instead of ErrorToReturn
}

// NewMockClient creates a new mock GitHub client
func NewMockClient() *MockClient {
	return &MockClient{
		Repositories: make(map[string]*github.Repository),
		Releases:     make(map[string][]*github.RepositoryRelease),
		Users:        make(map[string]*github.User),
		FileContents: make(map[string]*github.RepositoryContent),
		CreatedFiles: make(map[string][]byte),
		UpdatedFiles: make(map[string][]byte),
	}
}

// GetRepository fetches repository information from mock data
func (m *MockClient) GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error) {
	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}

	key := fmt.Sprintf("%s/%s", owner, repo)
	if repo, exists := m.Repositories[key]; exists {
		return repo, nil
	}

	return nil, fmt.Errorf("repository %s not found", key)
}

// GetRelease fetches a specific release from mock data
func (m *MockClient) GetRelease(ctx context.Context, owner, repo, tag string) (*github.RepositoryRelease, error) {
	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}

	key := fmt.Sprintf("%s/%s", owner, repo)
	releases, exists := m.Releases[key]
	if !exists {
		return nil, fmt.Errorf("no releases found for %s", key)
	}

	for _, release := range releases {
		if release.TagName != nil && *release.TagName == tag {
			return release, nil
		}
	}

	return nil, fmt.Errorf("release %s not found in %s", tag, key)
}

// ListReleases fetches all releases from mock data
func (m *MockClient) ListReleases(ctx context.Context, owner, repo string) ([]*github.RepositoryRelease, error) {
	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}

	key := fmt.Sprintf("%s/%s", owner, repo)
	releases, exists := m.Releases[key]
	if !exists {
		return []*github.RepositoryRelease{}, nil
	}

	return releases, nil
}

// CreateRelease creates a new release in mock data
func (m *MockClient) CreateRelease(ctx context.Context, owner, repo string, release *github.RepositoryRelease) (*github.RepositoryRelease, error) {
	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}

	key := fmt.Sprintf("%s/%s", owner, repo)
	if _, exists := m.Releases[key]; !exists {
		m.Releases[key] = []*github.RepositoryRelease{}
	}

	// Assign synthetic ID and URL for testing
	id := int64(len(m.Releases[key]) + 1)
	release.ID = &id
	htmlURL := fmt.Sprintf("https://github.com/%s/releases/tag/%s", key, release.GetTagName())
	release.HTMLURL = &htmlURL

	m.Releases[key] = append(m.Releases[key], release)
	return release, nil
}

// UploadReleaseAsset simulates uploading an asset to a release.
// If UploadError is set, it is returned instead of ErrorToReturn.
func (m *MockClient) UploadReleaseAsset(ctx context.Context, owner, repo string, releaseID int64, assetPath, contentType string) (*github.ReleaseAsset, error) {
	if m.UploadError != nil {
		return nil, m.UploadError
	}
	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}

	m.UploadedAssets = append(m.UploadedAssets, assetPath)

	name := assetPath
	asset := &github.ReleaseAsset{
		Name: &name,
	}

	return asset, nil
}

// GetAuthenticatedUser returns mock authenticated user
func (m *MockClient) GetAuthenticatedUser(ctx context.Context) (*github.User, error) {
	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}

	for _, user := range m.Users {
		return user, nil
	}

	return nil, fmt.Errorf("no authenticated user found")
}

// ForkRepository simulates forking a repository
func (m *MockClient) ForkRepository(ctx context.Context, owner, repo string) (*github.Repository, error) {
	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}

	originalKey := fmt.Sprintf("%s/%s", owner, repo)
	original, exists := m.Repositories[originalKey]
	if !exists {
		return nil, fmt.Errorf("repository %s not found", originalKey)
	}

	// Create a fork
	fork := &github.Repository{
		Name:        original.Name,
		FullName:    github.String(fmt.Sprintf("mockuser/%s", *original.Name)),
		Description: original.Description,
		Fork:        github.Bool(true),
	}

	m.Repositories["mockuser/"+*original.Name] = fork
	return fork, nil
}

// CreatePullRequest simulates creating a pull request
func (m *MockClient) CreatePullRequest(ctx context.Context, owner, repo string, pr *github.NewPullRequest) (*github.PullRequest, error) {
	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}

	pullRequest := &github.PullRequest{
		Title: pr.Title,
		Body:  pr.Body,
		Head:  &github.PullRequestBranch{Ref: pr.Head},
		Base:  &github.PullRequestBranch{Ref: pr.Base},
	}

	return pullRequest, nil
}

// SetError sets an error to be returned by all mock operations
func (m *MockClient) SetError(err error) {
	m.ErrorToReturn = err
}

// AddRepository adds a repository to mock data
func (m *MockClient) AddRepository(owner, repo string, repository *github.Repository) {
	key := fmt.Sprintf("%s/%s", owner, repo)
	m.Repositories[key] = repository
}

// AddRelease adds a release to mock data
func (m *MockClient) AddRelease(owner, repo string, release *github.RepositoryRelease) {
	key := fmt.Sprintf("%s/%s", owner, repo)
	if _, exists := m.Releases[key]; !exists {
		m.Releases[key] = []*github.RepositoryRelease{}
	}
	m.Releases[key] = append(m.Releases[key], release)
}

// AddUser adds a user to mock data
func (m *MockClient) AddUser(user *github.User) {
	m.Users[user.GetLogin()] = user
}

// GetFileContents retrieves file contents from mock data
func (m *MockClient) GetFileContents(ctx context.Context, owner, repo, path string) (*github.RepositoryContent, error) {
	if m.ContentsError != nil {
		return nil, m.ContentsError
	}
	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}

	key := fmt.Sprintf("%s/%s/%s", owner, repo, path)
	content, exists := m.FileContents[key]
	if !exists {
		return nil, fmt.Errorf("file %s not found in %s/%s: 404 Not Found", path, owner, repo)
	}
	return content, nil
}

// CreateFile simulates creating a file in a repository
func (m *MockClient) CreateFile(ctx context.Context, owner, repo, path, message string, content []byte) error {
	if m.ErrorToReturn != nil {
		return m.ErrorToReturn
	}

	key := fmt.Sprintf("%s/%s/%s", owner, repo, path)
	m.CreatedFiles[key] = content
	return nil
}

// UpdateFile simulates updating a file in a repository
func (m *MockClient) UpdateFile(ctx context.Context, owner, repo, path, message string, content []byte, sha string) error {
	if m.ErrorToReturn != nil {
		return m.ErrorToReturn
	}

	key := fmt.Sprintf("%s/%s/%s", owner, repo, path)
	m.UpdatedFiles[key] = content
	return nil
}

// AddFileContent adds file content to mock data for GetFileContents
func (m *MockClient) AddFileContent(owner, repo, path string, content *github.RepositoryContent) {
	key := fmt.Sprintf("%s/%s/%s", owner, repo, path)
	m.FileContents[key] = content
}
