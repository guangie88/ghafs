package main

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/google/go-github/v28/github"
	"golang.org/x/oauth2"
)

type ReleaseAssets struct {
	release *github.RepositoryRelease
	assets  []*github.ReleaseAsset
}

type ReleaseAssetsMap map[string]ReleaseAssets

func rasToDirents(rasm *ReleaseAssetsMap) []fuse.Dirent {
	dirents := make([]fuse.Dirent, len(*rasm))

	i := 0
	for tag, ras := range *rasm {
		dirents[i] = fuse.Dirent{
			Inode: uint64(ras.release.GetID()),
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

type Update struct{}

// ghaFS implements the GitHub Release Assets file system.
type ghaFS struct {
	owner string
	repo  string
	token string
	rasm  *ReleaseAssetsMap
}

func (state *ghaFS) Receive(actx actor.Context) {
	switch msg := actx.Message().(type) {

	case Update:
		// GitHub Set-up
		ctx := context.Background()

		var tc *http.Client

		if state.token != "" {
			ts := oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: state.token},
			)
			tc = oauth2.NewClient(ctx, ts)
		}

		client := github.NewClient(tc)
		releases, _, err := client.Repositories.ListReleases(ctx, state.owner, state.repo, nil)

		if err != nil {
			log.Fatal(err)
		}

		rasm := make(ReleaseAssetsMap)

		for _, release := range releases {
			assets, _, err := client.Repositories.ListReleaseAssets(ctx, state.owner, state.repo, release.GetID(), nil)

			if err != nil {
				log.Fatal(err)
			}

			rasm[release.GetTagName()] = ReleaseAssets{
				release,
				assets,
			}
		}

		state.rasm = &rasm
	}
}

func NewGhaFS(owner string, repo string, token string) ghaFS {
	return ghaFS{owner, repo, token, nil}
}

func (g ghaFS) Root() (fs.Node, error) {
	return root{rasm: g.rasm, token: g.token}, nil
}

// root implements both Node and Handle for the root directory.
type root struct {
	rasm  *ReleaseAssetsMap
	token *string
}

func (root) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 1
	a.Mode = os.ModeDir | 0555
	return nil
}

func (r root) Lookup(ctx context.Context, name string) (fs.Node, error) {
	for tag, ras := range *r.rasm {
		if name == tag {
			return tagDir{ras: &ras, token: r.token}, nil
		}
	}

	return nil, fuse.ENOENT
}

func (r root) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	return rasToDirents(r.rasm), nil
}

// tagDir implements both Node and Handle for the root directory.
type tagDir struct {
	ras   *ReleaseAssets
	token *string
}

func (t tagDir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = uint64(t.ras.release.GetID())
	a.Mode = os.ModeDir | 0555
	return nil
}

func (t tagDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	for _, asset := range t.ras.assets {
		if name == asset.GetName() {
			return assetFile{ra: asset, token: t.token}, nil
		}
	}
	return nil, fuse.ENOENT
}

func (t tagDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	return assetsToDirents(t.ras.assets), nil
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
