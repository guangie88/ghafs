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
	rasm  *ReleaseAssetsMap
	token *string
}

// root implements both Node and Handle for the root directory.
type root struct {
	rasm  *ReleaseAssetsMap
	token *string
}

// tagDir implements both Node and Handle for the root directory.
type tagDir struct {
	ras   *ReleaseAssets
	token *string
}

func NewGhaFS(rasm *ReleaseAssetsMap, token *string) ghaFS {
	return ghaFS{rasm, token}
}

func (g ghaFS) Root() (fs.Node, error) {
	return root{rasm: g.rasm, token: g.token}, nil
}

func (root) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 1
	a.Mode = os.ModeDir | 0555
	return nil
}

func (r root) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	return rasToDirents(r.rasm), nil
}

func (r root) Lookup(ctx context.Context, name string) (fs.Node, error) {
	for tag, ras := range *r.rasm {
		if name == tag {
			return tagDir{ras: &ras, token: r.token}, nil
		}
	}

	return nil, fuse.ENOENT
}

func (t tagDir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = uint64(t.ras.release.GetID())
	a.Mode = os.ModeDir | 0555
	return nil
}

func (t tagDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	return assetsToDirents(t.ras.assets), nil
}

func (t tagDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	for _, asset := range t.ras.assets {
		if name == asset.GetName() {
			return assetFile{ra: asset, token: t.token}, nil
		}
	}
	return nil, fuse.ENOENT
}

// assetFile implements both Node and Handle for the hello file.
type assetFile struct {
	ra    *github.ReleaseAsset
	token *string
}

func (f assetFile) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = uint64(f.ra.GetID())
	a.Mode = 0444
	a.Size = uint64(f.ra.GetSize())
	return nil
}

func (f assetFile) ReadAll(ctx context.Context) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", f.ra.GetURL(), nil)
	req.Header.Add("Accept", "application/octet-stream")

	if f.token != nil {
		req.Header.Add("Authorization", "token "+*f.token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	log.Printf("Asset URL: %v, Content-Length: %v", f.ra.GetURL(), resp.ContentLength)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
