package main

import (
	"context"

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
