package build

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

const gitPath = "https://github.com/%s"
const buildPath = "srv/www/%s-build-%s"

func buildExists(repoName, hash string) bool {
	var dir = fmt.Sprintf(buildPath, repoName, hash)
	_, err := os.Stat(dir)
	return !os.IsNotExist(err)
}

func dockerize(dirpath string) error {
	b, err := exec.Command("make", "-C", dirpath, "docker").Output()
	if len(b) > 0 {
		fmt.Println("exec resp:", string(b))
	}
	return err
}

func Build(repoName, hash string) error {
	if buildExists(repoName, hash) {
		return errors.New("project has already been built")
	}

	var dir = fmt.Sprintf(buildPath, repoName, hash)
	fmt.Println("setting up ", dir)
	os.MkdirAll(dir, 0777)
	//	defer os.RemoveAll(dir)

	r, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL: fmt.Sprintf(gitPath, repoName),
	})
	if err != nil {
		return err
	}

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	// ... checking out to commit
	if err = w.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(hash),
	}); err != nil {
		return err
	}

	return nil
	//return dockerize(dir)
}
