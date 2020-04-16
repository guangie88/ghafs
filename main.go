package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/google/go-github/v28/github"
	"golang.org/x/oauth2"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <mountpoint> <owner> <repo> [access-token]\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() < 3 || flag.NArg() > 4 {
		usage()
		os.Exit(1)
	}

	mountpoint := flag.Arg(0)
	owner := flag.Arg(1)
	repo := flag.Arg(2)
	oauth2Token := flag.Arg(3)

	// GitHub Set-up
	ctx := context.Background()

	var tc *http.Client
	if oauth2Token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: oauth2Token},
		)
		tc = oauth2.NewClient(ctx, ts)
	}

	client := github.NewClient(tc)
	rasm, err := getReleaseAssets(ctx, client, owner, repo)

	if err != nil {
		log.Fatal(err)
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

	// Prepare to pass around the token via reference
	var token *string
	if oauth2Token != "" {
		token = &oauth2Token
	}

	err = fs.Serve(c, NewGhaFS(rasm, token))
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}
