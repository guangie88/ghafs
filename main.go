package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/alexflint/go-arg"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/google/go-github/v28/github"
	"golang.org/x/oauth2"
)

type args struct {
	Mountpoint       string `arg:"positional,required"`
	Owner            string `arg:"positional,required"`
	Repo             string `arg:"positional,required"`
	AccessToken      string `arg:"--token" help:"GitHub access token for authorization"`
	AllowOther       bool   `arg:"--allow-other" help:"use FUSE allow_other mode (allow_root doesn't work, so not available)"`
	RefreshThreshold uint32 `arg:"--refresh" default:"30" help:"number of seconds that must elapsed before subsequent assets refresh can occur"`
}

func (args) Description() string {
	return "GitHub Release Assets FUSE CLI"
}

func (args) Version() string {
	return "ghafs 0.1.3"
}

func main() {
	var args args
	arg.MustParse(&args)

	// GitHub Set-up
	ctx := context.Background()

	var tc *http.Client
	if args.AccessToken != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: args.AccessToken},
		)
		tc = oauth2.NewClient(ctx, ts)
	}

	client := github.NewClient(tc)
	mgmt, err := makeReleaseMgmt(makeGhContext(ctx, client, args.Owner, args.Repo, time.Duration(args.RefreshThreshold)*time.Second))

	if err != nil {
		log.Fatal(err)
	}

	mountOptions := []fuse.MountOption{
		fuse.FSName("ghafs"),
		fuse.Subtype("ghafs"),
		fuse.LocalVolume(),
		fuse.VolumeName(fmt.Sprintf("GitHub Release Assets Filesystem for %s/%s", args.Owner, args.Repo)),
		fuse.ReadOnly(),
	}

	if args.AllowOther {
		mountOptions = append(mountOptions, fuse.AllowOther())
	}

	// FUSE Set-up
	c, err := fuse.Mount(
		args.Mountpoint,
		mountOptions...,
	)

	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()

	// Prepare to pass around the token via reference
	err = fs.Serve(c, newGhaFS(mgmt, &args.AccessToken))
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}
