package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/google/go-github/v28/github"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <mountpoint> <owner> <repo>\n", os.Args[0])
	flag.PrintDefaults()
}

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

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 3 {
		usage()
		os.Exit(1)
	}

	mountpoint := flag.Arg(0)
	owner := flag.Arg(1)
	repo := flag.Arg(2)

	// GitHub Set-up
	ctx := context.Background()
	client := github.NewClient(nil)
	releases, _, err := client.Repositories.ListReleases(ctx, owner, repo, nil)

	if err != nil {
		log.Fatal(err)
	}

	rasm := make(ReleaseAssetsMap)

	for _, release := range releases {
		assets, _, err := client.Repositories.ListReleaseAssets(ctx, owner, repo, release.GetID(), nil)

		if err != nil {
			log.Fatal(err)
		}

		rasm[release.GetTagName()] = ReleaseAssets{
			release,
			assets,
		}
	}

	// FUSE Set-up
	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("ghafs"),
		fuse.Subtype("ghafs"),
		fuse.LocalVolume(),
		fuse.VolumeName(fmt.Sprintf("GitHub Release Assets Filesystem for %s/%s", owner, repo)),
	)

	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()

	err = fs.Serve(c, newGhaFS(&rasm))
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}

// GhaFS implements the GitHub Release Assets file system.
type GhaFS struct {
	rasm *ReleaseAssetsMap
}

func newGhaFS(rasm *ReleaseAssetsMap) GhaFS {
	return GhaFS{rasm}
}

func (g GhaFS) Root() (fs.Node, error) {
	return RootDir{rasm: g.rasm}, nil
}

// RootDir implements both Node and Handle for the root directory.
type RootDir struct {
	rasm *ReleaseAssetsMap
}

func (RootDir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 1
	a.Mode = os.ModeDir | 0555
	return nil
}

func (r RootDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	for tag, ras := range *r.rasm {
		if name == tag {
			return TagDir{ras: &ras}, nil
		}
	}

	return nil, fuse.ENOENT
}

func (r RootDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	return rasToDirents(r.rasm), nil
}

// TagDir implements both Node and Handle for the root directory.
type TagDir struct {
	ras *ReleaseAssets
}

func (t TagDir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = uint64(t.ras.release.GetID())
	a.Mode = os.ModeDir | 0555
	return nil
}

func (t TagDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	for _, asset := range t.ras.assets {
		if name == asset.GetName() {
			return File{ra: asset}, nil
		}
	}
	return nil, fuse.ENOENT
}

func (t TagDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	return assetsToDirents(t.ras.assets), nil
}

// File implements both Node and Handle for the hello file.
type File struct {
	ra *github.ReleaseAsset
}

func (f File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = uint64(f.ra.GetID())
	a.Mode = 0444
	a.Size = uint64(f.ra.GetSize())
	return nil
}

func (f File) ReadAll(ctx context.Context) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", f.ra.GetURL(), nil)
	req.Header.Add("Accept", "application/octet-stream")

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
