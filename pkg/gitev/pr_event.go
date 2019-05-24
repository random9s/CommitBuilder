package gitev

import "fmt"

//Action symbolizes the pull request action type
type Action string

const (
	ACTION_ASSIGN        Action = "assigned"
	ACTION_UNASSIGN      Action = "unassigned"
	ACTION_REV_REQ       Action = "review_requested"
	ACTION_REV_REQ_REMOV Action = "review_request_removed"
	ACTION_LABEL         Action = "labeled"
	ACTION_UNLABEL       Action = "unlabeled"
	ACTION_OPEN          Action = "opened"
	ACTION_EDIT          Action = "edited"
	ACTION_CLOSE         Action = "closed"
	ACTION_READY_FOR_REV Action = "ready_for_review"
	ACTION_LOCK          Action = "locked"
	ACTION_UNLOCK        Action = "unlocked"
	ACTION_REOPEN        Action = "reopened"
	ACTION_SYNC          Action = "synchronize"
)

//GitUser ...
type GitUser struct {
	Login       string `json:"login"`
	ID          int    `json:"id"`
	NodeID      string `json:"node_id"`
	AvatarURL   string `json:"avatar_url"`
	GravatarURL string `json:"gravatar_id"`
	HTMLUrl     string `json:"html_url"`
	URL         string `json:"url"`
	Type        string `json:"type"`
	Admin       bool   `json:"site_admin"`
}

//RepoInfo ...
type RepoInfo struct {
	ID       int     `json:"id"`
	NodeID   string  `json:"node_id"`
	Name     string  `json:"name"`
	FullName string  `json:"full_name"`
	Private  bool    `json:"private"`
	Owner    GitUser `json:"owner"`
}

//HeadRef ...
type HeadRef struct {
	Label    string   `json:"label"`
	Ref      string   `json:"ref"`
	Sha      string   `json:"sha"`
	User     GitUser  `json:"user"`
	HTMLURL  string   `json:"html_url"`
	GitURL   string   `json:"git_url"`
	SSHURL   string   `json:"ssh_url"`
	CloneURL string   `json:"clone_url"`
	Repo     RepoInfo `json:"repo"`
}

//PullReq ...
type PullReq struct {
	URL     string  `json:"url"`
	ID      int     `json:"id"`
	NodeID  string  `json:"node_id"`
	HTMLUrl string  `json:"html_url"`
	Number  int     `json:"number"`
	State   string  `json:"state"`
	Locked  bool    `json:"locked"`
	Title   string  `json:"title"`
	User    GitUser `json:"user"`
	Head    HeadRef `json:"head"`
}

type State string

const (
	STATE_BUILDING State = "building"
	STATE_ACTIVE   State = "active"
	STATE_FAILED   State = "failed"
)

//PullReqEvent ...
type PullReqEvent struct {
	Action   Action  `json:"action"`
	PRNumber int     `json:"number"`
	PullReq  PullReq `json:"pull_request"`
	State    State   `json:"__local_state"`
	Loc      string  `json:"server_loc"`
}

func (pre *PullReqEvent) SetActive() {
	pre.State = STATE_ACTIVE
}

func (pre *PullReqEvent) SetBuilding() {
	pre.State = STATE_BUILDING
}

func (pre *PullReqEvent) SetFailed() {
	pre.State = STATE_FAILED
}

func (pre *PullReqEvent) SetBuildLoc(loc string) {
	pre.Loc = loc
}

func (pre *PullReqEvent) String() string {
	return fmt.Sprintf("Action: %s\nPR Number: %d\n\nBody: %#v\n", pre.Action, pre.PRNumber, pre.PullReq)
}
