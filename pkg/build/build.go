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

func dockerize(dirpath, containerName string) (string, error) {
	fmt.Println("DOCKERIZIN")
	var makefile = fmt.Sprintf("%s/Makefile", dirpath)
	if _, err := os.Stat(makefile); os.IsNotExist(err) {
		fmt.Println("cannot find makefile")
		return "", errors.New("could not locate Makefile in project")
	}
	fmt.Println("MAKEFILE EXISTS")

	port := network.NextAvailablePort()
	if port == 0 {
		fmt.Println("cannot find port")
		return "", errors.New("no available port to run docker container")
	}
	fmt.Println("PORT", port)
	var loc = fmt.Sprintf("http://ec2-34-215-250-175.us-west-2.compute.amazonaws.com:%d", port)

	cmd := exec.Command(
		"make",
		fmt.Sprintf("PORT=%d", port),
		fmt.Sprintf("CONTAINER=%s", containerName),
		"-C", dirpath,
		"docker",
	)
	fmt.Println("CREATED COMMAND")

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	cmd.Start()
	b, _ := ioutil.ReadAll(stderr)
	sb, _ := ioutil.ReadAll(stdout)
	cmd.Wait()
	fmt.Println("OUTPUT:", string(sb))
	if len(b) > 0 {
		fmt.Println("ugh", string(b))
		return "", errors.New("could not run makefile: " + string(b))
	}

	return loc, nil
}

func Build(pre *gitev.PullReqEvent, containerName string) (string, error) {
	var dir = fmt.Sprintf(buildPath, pre.PullReq.Head.Repo.Name, pre.PullReq.Head.Sha)
	fmt.Println("check dir", dir)
	if buildExists(dir) {
		return "", errors.New("project has already been built")
	}
	os.MkdirAll(dir, 0777)
	defer os.RemoveAll(dir)
	fmt.Println("dir is created", dir)

	fmt.Println(fmt.Sprintf(gitPath, pre.PullReq.Head.Repo.FullName))
	r, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL: fmt.Sprintf(gitPath, pre.PullReq.Head.Repo.FullName),
	})
	fmt.Printf("CLONE %#v\n", r)
	if err != nil {
		fmt.Println("ERROR?", err)
		return "", err
	}
	fmt.Println("cloning and boning")

	w, err := r.Worktree()
	fmt.Printf("WORKTREE %#v\n", w)
	if err != nil {
		fmt.Println("ERROR?", err)
		return "", err
	}
	fmt.Println("get work tree")

	if err = w.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(pre.PullReq.Head.Sha),
	}); err != nil {
		fmt.Println("ERROR?", err)
		return "", err
	}
	fmt.Println("checked out hash")

	return dockerize(dir, containerName)
}
