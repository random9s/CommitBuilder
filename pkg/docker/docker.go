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
	var pullReqID = pre.PRNumber
	var projName = pre.PullReq.Head.Repo.Name
	var commitSha = pre.PullReq.Head.Sha[10:]
	return fmt.Sprintf("%s-%d-%s", projName, pullReqID, commitSha)
}

//PRContainer ...
func PRContainer(pre *gitev.PullReqEvent) (string, error) {
	containers, err := ListContainers()
	if err != nil {
		return "", err
	}

	var projName = pre.PullReq.Head.Repo.Name
	var pullReqID = pre.PRNumber
	var pref = fmt.Sprintf("%s-%s", projName, pullReqID)

	var container string
	for _, c := range containers {
		if strings.HasPrefix(c, pref) {
			container = c
			break
		}
	}

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
