package main

import (
	"context"
	"sync"

	"github.com/google/go-github/v28/github"
)

// GhContext contains the necessary inputs to invoke the Gitub library
type GhContext struct {
	ctx    context.Context
	client *github.Client
	owner  string
	repo   string
}

// ReleaseMgmt forms the root level to be able to generate the entire release
// assets content with the given GitHub context
type ReleaseMgmt struct {
	ghc      *GhContext
	releases *ReleasesWrap
}

// ReleasesWrap forms the lazy layer to fetch the actual releases
type ReleasesWrap struct {
	ghc *GhContext

	tagReleases map[string]*Release
	m           sync.Mutex
}

// Release contains the mapping between the actual release and the assets lazy
// layer
type Release struct {
	ghc     *GhContext
	content *github.RepositoryRelease
	assets  *AssetsWrap
}

// AssetsWrap forms the lazy layer to fetch the actual assets based on given
// release
type AssetsWrap struct {
	ghc     *GhContext
	release *github.RepositoryRelease

	assets []*github.ReleaseAsset
	m      sync.Mutex
}

func makeGhContext(ctx context.Context, client *github.Client, owner string, repo string) *GhContext {
	return &GhContext{ctx, client, owner, repo}
}

func makeReleaseMgmt(ghc *GhContext) *ReleaseMgmt {
	return &ReleaseMgmt{ghc, makeReleasesWrap(ghc)}
}

func makeReleasesWrap(ghc *GhContext) *ReleasesWrap {
	w := &ReleasesWrap{}
	w.ghc = ghc
	w.tagReleases = make(map[string]*Release)
	return w
}

func makeRelease(ghc *GhContext, release *github.RepositoryRelease) *Release {
	return &Release{ghc, release, makeAssetsWrap(ghc, release)}
}

func makeAssetsWrap(ghc *GhContext, release *github.RepositoryRelease) *AssetsWrap {
	w := &AssetsWrap{}
	w.ghc = ghc
	w.release = release
	w.assets = []*github.ReleaseAsset{}
	return w
}

// refreshImpl to be only internally only, to be used by mutex wrapping methods
func (w ReleasesWrap) refreshImpl() (map[string]*Release, error) {
	releases, _, err := w.ghc.client.Repositories.ListReleases(
		w.ghc.ctx, w.ghc.owner, w.ghc.repo, nil)

	if err != nil {
		return nil, err
	}

	for _, release := range releases {
		w.tagReleases[release.GetTagName()] = makeRelease(w.ghc, release)
	}

	return w.tagReleases, nil
}

func (w ReleasesWrap) refresh() (map[string]*Release, error) {
	w.m.Lock()
	defer w.m.Unlock()
	return w.refreshImpl()
}

func (w ReleasesWrap) get() (map[string]*Release, error) {
	w.m.Lock()
	defer w.m.Unlock()

	if len(w.tagReleases) == 0 {
		return w.refreshImpl()
	}

	return w.tagReleases, nil
}

// refreshImpl to be only internally only, to be used by mutex wrapping methods
func (w AssetsWrap) refreshImpl() ([]*github.ReleaseAsset, error) {
	assets, _, err := w.ghc.client.Repositories.ListReleaseAssets(
		w.ghc.ctx,
		w.ghc.owner,
		w.ghc.repo,
		w.release.GetID(),
		nil)

	if err != nil {
		return nil, err
	}

	w.assets = assets
	return w.assets, nil
}

func (w AssetsWrap) refresh() ([]*github.ReleaseAsset, error) {
	w.m.Lock()
	defer w.m.Unlock()
	return w.refreshImpl()
}

func (w AssetsWrap) get() ([]*github.ReleaseAsset, error) {
	w.m.Lock()
	defer w.m.Unlock()

	if len(w.assets) == 0 {
		return w.refreshImpl()
	}

	return w.assets, nil
}
