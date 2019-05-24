package build

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/random9s/CommitBuilder/pkg/gitev"
	"github.com/random9s/CommitBuilder/pkg/network"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

const gitPath = "https://github.com/%s"
const buildPath = "/srv/www/%s-build-%s"

func buildExists(dir string) bool {
	_, err := os.Stat(dir)
	return !os.IsNotExist(err)
}

func dockerize(dirpath, containerName string) error {
	var makefile = fmt.Sprintf("%s/Makefile", dirpath)
	if _, err := os.Stat(makefile); os.IsNotExist(err) {
		return errors.New("could not locate Makefile in project")
	}

	port := network.NextAvailablePort()
	if port == 0 {
		return errors.New("no available port to run docker container")
	}

	cmd := exec.Command(
		"make",
		fmt.Sprintf("PORT=%d", port),
		fmt.Sprintf("CONTAINER=%s", containerName),
		"-C", dirpath,
		"docker",
	)
	stderr, _ := cmd.StderrPipe()
	cmd.Start()
	b, _ := ioutil.ReadAll(stderr)
	cmd.Wait()
	if len(b) > 0 {
		return errors.New("could not run makefile: " + string(b))
	}

	return nil
}

func Build(pre *gitev.PullReqEvent, containerName string) error {
	var dir = fmt.Sprintf(buildPath, pre.PullReq.Head.Repo.Name, pre.PullReq.Head.Sha)
	if buildExists(dir) {
		return errors.New("project has already been built")
	}
	os.MkdirAll(dir, 0777)
	defer os.RemoveAll(dir)

	fmt.Println("cloning dir")

	r, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL: fmt.Sprintf(gitPath, pre.PullReq.Head.Repo.FullName),
	})
	if err != nil {
		return err
	}

	fmt.Println("setting up work tree")

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	fmt.Println("checking out current commit")

	if err = w.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(pre.PullReq.Head.Sha),
	}); err != nil {
		return err
	}

	return dockerize(dir, containerName)
}
