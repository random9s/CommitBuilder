package main

//STAGING

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/random9s/cinder/logger"
	logfmt "github.com/random9s/cinder/logger/format"

	"github.com/random9s/CommitBuilder/cmd/commitbuilder/routes"
	"github.com/random9s/CommitBuilder/pkg/middleware"
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
	var port int64
	var debug bool
	flag.Int64Var(&port, "port", 8080, "port to listen")
	flag.BoolVar(&debug, "debug", false, "debug mode")
	flag.Parse()

	var srv http.Server
	srv.Addr = fmt.Sprintf(":%d", port)

	/*
	 * CREATE ACCESS AND ERROR LOGS
	 */
	var accessPath, errorPath = fmt.Sprintf("/var/log/prserver/access/access-%d", time.Now().Unix()), fmt.Sprintf("/var/log/prserver/error/error-%d", time.Now().Unix())
	var symAccess, symError = "/var/log/prserver/access/sym-access", "/var/log/prserver/error/sym-error"
	if debug {
		accessPath, errorPath = fmt.Sprintf("log/prserver/access/access-%d", time.Now().Unix()), fmt.Sprintf("log/prserver/error/error-%d", time.Now().Unix())
		symAccess, symError = "log/prserver/access/sym-access", "log/prserver/error/sym-error"
	}
	access, erro := initLog(accessPath), initLog(errorPath)
	if access.Size() == 0 {
		access.Write(logfmt.NewDirective("Proxy", "1.0", "call api", "Date", "Time", "File", "IP", "Method", "URI", "Time Taken", "Status", "Bytes").ToBytes())
	}
	if erro.Size() == 0 {
		erro.Write(logfmt.NewDirective("Proxy", "1.0", "call api", "Date", "Time", "File", "Error").ToBytes())
	}
	os.Remove(symAccess)
	os.Remove(symError)
	os.Symlink(accessPath, symAccess)
	os.Symlink(errorPath, symError)

	var prStateDir = "/srv/www/prstates"
	if debug {
		prStateDir = "srv/www/prstates"
	}
	if _, err := os.Stat(prStateDir); os.IsNotExist(err) {
		os.MkdirAll(prStateDir, 0777)
	}

	/*
	 * SETUP ROUTER AND MIDDLEWARE FOR LOGGIN REQS
	 */
	r := mux.NewRouter().StrictSlash(true)

	logReq := middleware.AccessLogger(access)
	indexGet := routes.IndexGet(erro)
	indexPost := routes.IndexPost(erro, prStateDir)

	var pingCh = make(chan bool)
	var infoCh = make(chan []byte)
	go func() {
		for {
			select {
			case _, ok := <-pingCh:
				if !ok {
					fmt.Println("time to die")
					return
				}

				var JSON string
				files, _ := ioutil.ReadDir(prStateDir)
				for _, file := range files {
					b, _ := ioutil.ReadFile(fmt.Sprintf("%s/%s", prStateDir, file.Name()))
					JSON += fmt.Sprintf("%s,", string(b))
				}
				JSON = strings.TrimRight(JSON, ",")
				infoCh <- []byte(fmt.Sprintf(`[%s]`, JSON))
			}
		}
	}()
	indexSocket := routes.IndexWebSocketServer(erro, pingCh, infoCh)

	r.Methods("GET").Path("/").Name("index").Handler(logReq(indexGet))
	r.Methods("POST").Path("/").Name("pullreq").Handler(logReq(indexPost))
	r.Methods("GET").Path("/prinfo").Name("Pull Request Information").Handler(logReq(indexSocket))
	r.PathPrefix("/assets/").Handler(
		http.StripPrefix("/assets/", http.FileServer(http.Dir("assets/"))),
	)

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("page not found"))
	})
	srv.Handler = r

	/*
	 * SETUP GRACEFUL SERVER SHUTDOWN
	 */
	shutdown := make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func(srv *http.Server, logs ...logger.Logger) {
		select {
		case s := <-c:
			for _, l := range logs {
				l.Error("sig caught:", s)
				l.GzipClose()
			}

			if err := srv.Shutdown(context.Background()); err != nil {
				log.Printf("HTTP server Shutdown: %v", err)
			}

			close(infoCh)
			close(shutdown)
		}
	}(&srv, access, erro)

	fmt.Printf("listening at http://127.0.0.1%s...\n", srv.Addr)
	fmt.Println(srv.ListenAndServe())
	<-shutdown
}
