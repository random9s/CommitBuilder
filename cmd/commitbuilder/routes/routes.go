package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/gorilla/websocket"
	"github.com/random9s/CommitBuilder/pkg/build"
	"github.com/random9s/CommitBuilder/pkg/docker"
	"github.com/random9s/CommitBuilder/pkg/gitev"
	"github.com/random9s/cinder/logger"
)

func loadTemplate(w http.ResponseWriter, name string, htmlPaths ...string) {
	var data = make(map[string]interface{})

	err := template.Must(template.New(name+".html").ParseFiles(htmlPaths...)).ExecuteTemplate(w, name, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

//IndexWebSocketServer is used to poll for info about new PRs
func IndexWebSocketServer(errLog logger.Logger, ping chan bool, info chan []byte) http.Handler {
	var upgrader = websocket.Upgrader{}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("upgrade err:", err)
			return
		}
		defer c.Close()

		for {
			mt, _, err := c.ReadMessage()
			if err != nil {
				errLog.Error(err)
				break
			}

			ping <- true
			msg, ok := <-info
			if !ok {
				close(ping)
				return
			}

			if err = c.WriteMessage(mt, msg); err != nil {
				errLog.Error(err)
				break
			}
		}
	})
}

//IndexGet ...
func IndexGet(errLog logger.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Server-Status", strconv.Itoa(http.StatusOK))
		loadTemplate(w, "index.html", "assets/html/index.html")
	})
}

//IndexPost ...
func IndexPost(errLog logger.Logger, prStateDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var resp = []byte("forbidden\n")
		var status = strconv.Itoa(http.StatusForbidden)
		if r.URL.Query().Get("k") == "8cAzktzWjYSHNFpCYN3dP23UxkHJ7C8P" {
			resp = []byte("failure\n")
			status = strconv.Itoa(http.StatusInternalServerError)

			b, err := ioutil.ReadAll(r.Body)
			r.Body.Close()
			if err != nil {
				errLog.Error(err)
				w.Header().Set("X-Server-Status", status)
				w.Header().Set("Content-Length", strconv.Itoa(len(resp)))
				w.Write(resp)
				return
			}

			if len(b) == 0 {
				errLog.Error(errors.New("bad request: missing body"))
				resp = []byte("bad request: missing body")
				w.Header().Set("X-Server-Status", strconv.Itoa(http.StatusBadRequest))
				w.Header().Set("Content-Length", strconv.Itoa(len(resp)))
				w.Write(resp)
				return
			}

			var pre = new(gitev.PullReqEvent)
			err = json.Unmarshal(b, pre)
			if err != nil {
				errLog.Error(err)
				w.Header().Set("X-Server-Status", strconv.Itoa(http.StatusBadRequest))
				w.Header().Set("Content-Length", strconv.Itoa(len(resp)))
				w.Write(resp)
				return
			}

			go start(pre, prStateDir, errLog)

			resp = []byte("success\n")
			status = strconv.Itoa(http.StatusOK)
		}

		w.Header().Set("X-Server-Status", status)
		w.Header().Set("Content-Length", strconv.Itoa(len(resp)))
		w.Write(resp)
	})
}

func start(pre *gitev.PullReqEvent, dir string, errLog logger.Logger) {
	var stateFile = fmt.Sprintf("%s/%s-%d", dir, strings.ToLower(pre.PullReq.Head.Repo.Name), pre.PRNumber)
	fp, err := os.OpenFile(stateFile, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		errLog.Error(fmt.Errorf("err writing state file: %s", err.Error()))
		return
	}
	defer fp.Close()

	pre.SetBuilding()
	writeToDisk(pre, fp, errLog)

	loc, err := initializeEvent(pre, stateFile)
	if err != nil {
		errLog.Error(fmt.Errorf("err running action %s: %s", pre.Action, err.Error()))
		pre.SetFailed()
		writeToDisk(pre, fp, errLog)
		return
	}

	pre.SetActive()
	pre.SetBuildLoc(loc)
	writeToDisk(pre, fp, errLog)
}

func writeToDisk(pre *gitev.PullReqEvent, fp *os.File, errLog logger.Logger) {
	fp.Truncate(0)
	fp.Seek(0, 0)

	b, _ := json.Marshal(pre)
	if _, err := fp.Write(b); err != nil {
		errLog.Error(fmt.Errorf("err writing state file: %s", err.Error()))
	}

	fp.Sync()
}

func initializeEvent(pre *gitev.PullReqEvent, stateFile string) (string, error) {
	var name = docker.PRContainerName(pre)
	var serverLoc string
	var err error

	switch pre.Action {
	case gitev.ACTION_SYNC, gitev.ACTION_EDIT:
		fmt.Println("SYNC OR EDIT ACTION PERFORMED")
		if runningContainer, _ := docker.PRContainer(pre); runningContainer != "" {
			if err = docker.StopContainer(runningContainer); err != nil {
				break
			}
		}
		serverLoc, err = build.Build(pre, name)
	case gitev.ACTION_OPEN, gitev.ACTION_REOPEN:
		fmt.Println("OPEN OR REOPEN ACTION PERFORMED")
		serverLoc, err = build.Build(pre, name)
	case gitev.ACTION_CLOSE:
		fmt.Println("CLOSE ACTION PERFORMED")
		os.Remove(stateFile)
		err = docker.StopContainer(name)
	default:
		fmt.Println("NO ACTION FOR :", pre.Action)
	}

	return serverLoc, err
}
