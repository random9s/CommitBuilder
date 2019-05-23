package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/random9s/cinder/logger"
	logfmt "github.com/random9s/cinder/logger/format"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

func exitOnErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func initLog(path string) logger.Logger {
	l, err := logger.New(path)
	exitOnErr(err)
	return l
}

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "debug mode")
	flag.Parse()

	/*
	 * CREATE ACCESS AND ERROR LOGS
	 */
	var errorPath = fmt.Sprintf("/var/log/commitbuilder/error/error-%d", time.Now().Unix())
	var symError = "/var/log/commitbuilder/error/sym-error"
	if debug {
		errorPath = fmt.Sprintf("log/commitbuilder/error/error-%d", time.Now().Unix())
		symError = "log/commitbuilder/error/sym-error"
	}
	erro := initLog(errorPath)
	if erro.Size() == 0 {
		erro.Write(logfmt.NewDirective("Commit Builder", "1.0", "builds latest commit as docker container", "Date", "Time", "File", "Error").ToBytes())
	}
	os.Remove(symError)
	os.Symlink(errorPath, symError)

	/*
	 * SETUP SHUTDOWN LOGIC
	 */
	c := make(chan os.Signal, 1)
	signal.Notify(c)
	shutdown := make(chan struct{})

	/*
	 * Drop it like its hot
	 */
	dir, err := ioutil.TempDir("", "clone-temp")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("dir at ", dir)

	go func(dirname string, logs ...logger.Logger) {
		select {
		case s := <-c:
			for _, l := range logs {
				l.Error("sig caught:", s)
				l.GzipClose()
			}

			os.RemoveAll(dirname)
			close(shutdown)
			os.Exit(0)
		}
	}(dir, erro)

	r, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL: "https://github.com/random9s/CommitBuilder",
	})
	if err != nil {
		log.Fatal(err)
	}

	var hashes = make(map[string]uint)
	var t = time.NewTicker(time.Second * 5)

	for _ = range t.C {
		w, err := r.Worktree()
		if err != nil {
			continue
		}
		w.Pull(&git.PullOptions{RemoteName: "origin"})

		ref, err := r.Head()
		if err != nil {
			continue
		}

		// ... retrieves the commit history
		cIter, err := r.Log(&git.LogOptions{From: ref.Hash()})
		if err != nil {
			continue
		}
		// ... just iterates over the commits, printing it
		if err = cIter.ForEach(func(c *object.Commit) error {
			if time.Since(c.Author.When) >= (time.Hour * 24) {
				return nil
			}

			var has = c.Hash.String()
			if _, ok := hashes[has]; !ok {
				hashes[has] = 1
				setupBuild(has)
			}

			return nil
		}); err != nil {
			continue
		}
	}

	os.RemoveAll(dir)
}
func buildExists(hash string) bool {
	var dir = fmt.Sprintf("srv/www/build-%s", hash)
	_, err := os.Stat(dir)
	return !os.IsNotExist(err)
}

func setupBuild(hash string) {
	if buildExists(hash) {
		return
	}

	var dir = fmt.Sprintf("srv/www/build-%s", hash)
	fmt.Println("setting up ", dir)
	os.MkdirAll(dir, 0777)

	r, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL: "https://github.com/random9s/CommitBuilder",
	})
	if err != nil {
		log.Fatal(err)
	}

	w, err := r.Worktree()
	if err != nil {
		fmt.Println(err)
		return
	}

	// ... checking out to commit
	if err = w.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(hash),
	}); err != nil {
		fmt.Println(err)
		return
	}
}
