package main

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/google/go-github/v28/github"
)

// ghaFS implements the GitHub Release Assets file system.
type ghaFS struct {
	mgmt  *ReleaseMgmt
	token *string
}

// root implements both Node and Handle for the root directory.
type root struct {
	mgmt  *ReleaseMgmt
	token *string
}

// tagDir implements both Node and Handle for the root directory.
type tagDir struct {
	assets *AssetsWrap
	token  *string
}

// assetFile implements both Node and Handle for the hello file.
type assetFile struct {
	asset *github.ReleaseAsset
	token *string
}

func newGhaFS(mgmt *ReleaseMgmt, token *string) ghaFS {
	return ghaFS{mgmt, token}
}

func (g ghaFS) Root() (fs.Node, error) {
	return root{mgmt: g.mgmt, token: g.token}, nil
}

func (root) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 1
	a.Mode = os.ModeDir | 0775
	return nil
}

func (r root) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	releases, err := r.mgmt.releases.refresh()

	if err != nil {
		return nil, err
	}

	return releasesToDirents(releases), nil
}

func (r root) Lookup(ctx context.Context, name string) (fs.Node, error) {
	releases, err := r.mgmt.releases.get()

	if err != nil {
		return nil, err
	}

	for tag, release := range releases {
		if name == tag {
			return tagDir{assets: release.assets, token: r.token}, nil
		}
	}

	return nil, fuse.ENOENT
}

func (t tagDir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = uint64(t.assets.release.GetID())
	a.Mode = os.ModeDir | 0775
	return nil
}

func (t tagDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	assets, err := t.assets.refresh()

	if err != nil {
		return nil, err
	}

	return assetsToDirents(assets), nil
}

func (t tagDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	assets, err := t.assets.refresh()

	if err != nil {
		return nil, err
	}

	for _, asset := range assets {
		if name == asset.GetName() {
			return assetFile{asset: asset, token: t.token}, nil
		}
	}
	return nil, fuse.ENOENT
}

func (f assetFile) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = uint64(f.asset.GetID())
	a.Mode = 0664
	a.Size = uint64(f.asset.GetSize())
	return nil
}

func (f assetFile) ReadAll(ctx context.Context) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", f.asset.GetURL(), nil)
	req.Header.Add("Accept", "application/octet-stream")

	if f.token != nil {
		req.Header.Add("Authorization", "token "+*f.token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	log.Printf("Asset URL: %v, Content-Length: %v", f.asset.GetURL(), resp.ContentLength)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
