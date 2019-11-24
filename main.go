package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/AsynkronIT/protoactor-go/actor"
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

	context := actor.EmptyRootContext
	props := actor.PropsFromProducer(func() actor.Actor { return &GitHubActor{} })
	pid := context.Spawn(props)

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
	err = fs.Serve(c, NewGhaFS(oauth2Token))
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}
