package main

import (
	"context"
	"sync"

	"github.com/google/go-github/v28/github"
)

type ReleaseAssets struct {
	release *github.RepositoryRelease
	assets  []*github.ReleaseAsset
}

type ReleaseAssetsMap map[string]ReleaseAssets

func getReleaseAssets(ctx context.Context, client *github.Client, owner string, repo string) (*ReleaseAssetsMap, error) {
	rasm := make(ReleaseAssetsMap)

	releases, _, err := client.Repositories.ListReleases(ctx, owner, repo, nil)

	if err != nil {
		return nil, err
	}

	for _, release := range releases {
		assets, _, err := client.Repositories.ListReleaseAssets(ctx, owner, repo, release.GetID(), nil)

		if err != nil {
			return nil, err
		}

		rasm[release.GetTagName()] = ReleaseAssets{
			release,
			assets,
		}
	}

	return &rasm, nil
}

type ReleaseAssetsMgmt struct {
	ctx    context.Context
	client *github.Client
	owner  string
	repo   string

	current *ReleaseAssetsMap
	m       sync.Mutex
}

func makeReleaseAssetsMgmt(ctx context.Context, client *github.Client, owner string, repo string) (*ReleaseAssetsMgmt, error) {
	var mgmt ReleaseAssetsMgmt
	mgmt.ctx = ctx
	mgmt.client = client
	mgmt.owner = owner
	mgmt.repo = repo

	_, err := mgmt.refresh()
	return &mgmt, err
}

func (mgmt *ReleaseAssetsMgmt) refresh() (*ReleaseAssetsMap, error) {
	mgmt.m.Lock()
	defer mgmt.m.Unlock()

	var err error
	mgmt.current, err = getReleaseAssets(mgmt.ctx, mgmt.client, mgmt.owner, mgmt.repo)

	// See getCurrent for comments about correctness
	return mgmt.current, err
}

func (mgmt *ReleaseAssetsMgmt) getCurrent() *ReleaseAssetsMap {
	// For simplicity, we assume readonly access to ReleaseAssetsMgmt
	// so that we do not need to lock and return a cloned ReleaseAssetsMap
	// for correctness
	mgmt.m.Lock()
	defer mgmt.m.Unlock()
	return mgmt.current
}
