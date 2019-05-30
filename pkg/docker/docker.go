package docker

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/random9s/CommitBuilder/pkg/gitev"
)

//ListContainers ...
func ListContainers() ([]string, error) {
	b, err := exec.Command(
		"docker",
		"ps",
		"-a",
		"--format={{.Names}}",
	).Output()
	return strings.Split(string(b), "\n"), err
}

func PRContainerName(pre *gitev.PullReqEvent) string {
	var projName = pre.PullReq.Head.Repo.Name
	if len(projName) > 10 {
		projName = projName[:10]
	}
	var commitSha = pre.PullReq.Head.Sha
	if len(commitSha) > 10 {
		commitSha = commitSha[:10]
	}

	return strings.ToLower(fmt.Sprintf("%s-%d-%s", projName, pre.PRNumber, commitSha))
}

func PRContainerPrefix(pre *gitev.PullReqEvent) string {
	var projName = pre.PullReq.Head.Repo.Name
	if len(projName) > 10 {
		projName = projName[:10]
	}
	return strings.ToLower(fmt.Sprintf("%s-%d", projName, pre.PRNumber))
}

//PRContainer ...
func PRContainer(pre *gitev.PullReqEvent) (string, error) {
	containers, err := ListContainers()
	if err != nil {
		fmt.Println("super duper error", err)
		return "", err
	}

	var container string
	var pref = PRContainerPrefix(pre)
	fmt.Println("PREFIX: ", pref)

	for _, c := range containers {
		if strings.HasPrefix(c, pref) {
			container = c
			break
		}
	}

	fmt.Println("returning contianer: ", container)
	return container, nil
}

//StopContainer ...
func StopContainer(container string) error {
	b, err := exec.Command(
		"docker",
		"stop",
		container,
	).Output()
	if err != nil {
		return err
	}

	if len(b) > 0 {
		fmt.Println(string(b))
	}
	return nil
}
