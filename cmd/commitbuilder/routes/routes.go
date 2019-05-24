package routes

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
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
				fmt.Println("pack it in, shut it down")
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
				resp = []byte("bad request: missing body")
				w.Header().Set("X-Server-Status", strconv.Itoa(http.StatusBadRequest))
				w.Header().Set("Content-Length", strconv.Itoa(len(resp)))
				w.Write(resp)
				return
			}

			var pre = new(gitev.PullReqEvent)
			err = json.Unmarshal(b, pre)
			if err != nil {
				w.Header().Set("X-Server-Status", strconv.Itoa(http.StatusBadRequest))
				w.Header().Set("Content-Length", strconv.Itoa(len(resp)))
				w.Write(resp)
				return
			}
			pre.SetBuilding()

			preBytes, _ := json.Marshal(pre)
			var stateFile = fmt.Sprintf("%s/%s-%d", prStateDir, pre.PullReq.Head.Repo.Name, pre.PRNumber)
			fmt.Println("setting up state file: ", stateFile)
			err = ioutil.WriteFile(stateFile, preBytes, 0777)
			if err != nil {
				fmt.Println("err writing state file", err)
			}

			err = initializePREvent(pre)
			if err != nil {
				fmt.Println("Initialization error:", err)
				w.Header().Set("X-Server-Status", strconv.Itoa(http.StatusBadRequest))
				w.Header().Set("Content-Length", strconv.Itoa(len(resp)))
				w.Write(resp)
				return
			}

			fmt.Println("successfully completed action")
			resp = []byte("success\n")
			status = strconv.Itoa(http.StatusOK)
		}

		w.Header().Set("X-Server-Status", status)
		w.Header().Set("Content-Length", strconv.Itoa(len(resp)))
		w.Write(resp)
	})
}

func initializePREvent(pre *gitev.PullReqEvent) error {
	var name = docker.PRContainerName(pre)
	var err error

	switch pre.Action {
	case gitev.ACTION_SYNC, gitev.ACTION_EDIT:
		fmt.Println("SYNC OR EDIT ACTION PERFORMED")
		if runningContainer, _ := docker.PRContainer(pre); runningContainer != "" {
			fmt.Println("running container name is", runningContainer)
			if err = docker.StopContainer(runningContainer); err != nil {
				break
			}
			fmt.Println("shut down running container")
		}
		err = build.Build(pre, name)
	case gitev.ACTION_OPEN, gitev.ACTION_REOPEN:
		fmt.Println("OPEN OR REOPEN ACTION PERFORMED")
		err = build.Build(pre, name)
	case gitev.ACTION_CLOSE:
		fmt.Println("CLOSE ACTION PERFORMED")
		err = docker.StopContainer(name)
	default:
		fmt.Println("NO ACTION FOR :", pre.Action)
	}

	return err
}
