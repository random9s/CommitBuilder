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
			fmt.Println(err)
			continue
		}
		err = w.Pull(&git.PullOptions{RemoteName: "origin"})
		if err != nil {
			fmt.Println(err)
			continue
		}
		ref, err := r.Head()
		if err != nil {
			fmt.Println(err)
			continue
		}
		commit, err := r.CommitObject(ref.Hash())
		if err != nil {
			fmt.Println(err)
			continue
		}
		var has = commit.Hash.String()
		if _, ok := hashes[has]; !ok {
			fmt.Println("new hash found", has)
			hashes[has] = 1
		}
	}

	os.RemoveAll(dir)
}
