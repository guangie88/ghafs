package main

import (
	"bazil.org/fuse"
	"github.com/google/go-github/v28/github"
)

func releasesToDirents(tagReleases map[string]*Release) []fuse.Dirent {
	dirents := make([]fuse.Dirent, len(tagReleases))

	i := 0
	for tag, release := range tagReleases {
		dirents[i] = fuse.Dirent{
			Inode: uint64(release.content.GetID()),
			Name:  tag,
			Type:  fuse.DT_Dir,
		}
		i++
	}
	return dirents
}

func assetsToDirents(assets []*github.ReleaseAsset) []fuse.Dirent {
	dirents := make([]fuse.Dirent, len(assets))

	for i, asset := range assets {
		dirents[i] = fuse.Dirent{
			Inode: uint64(asset.GetID()),
			Name:  asset.GetName(),
			Type:  fuse.DT_File,
		}
	}
	return dirents
}
