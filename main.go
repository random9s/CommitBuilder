package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/random9s/cinder/logger"
	logfmt "github.com/random9s/cinder/logger/format"
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

func indexGet(errLog logger.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var resp = []byte("hello, is it me you're looking for?\n")
		var status, conLen = strconv.Itoa(http.StatusOK), strconv.Itoa(len(resp))
		w.Header().Set("X-Server-Status", status)
		w.Header().Set("Content-Length", conLen)
		w.Write(resp)
	})
}
func indexPost(errLog logger.Logger) http.Handler {
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

			fmt.Printf("BODY: %s\n", string(b))

			resp = []byte("success\n")
			status = strconv.Itoa(http.StatusOK)
		}

		w.Header().Set("X-Server-Status", status)
		w.Header().Set("Content-Length", strconv.Itoa(len(resp)))
		w.Write(resp)
	})
}

func accessLogger(l logger.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			now := time.Now()

			defer func(req *http.Request, start time.Time) {
				code, _ := strconv.ParseInt(w.Header().Get("X-Server-Status"), 10, 64)
				bytes, _ := strconv.ParseInt(w.Header().Get("Content-Length"), 10, 64)

				//Create new log entry
				var entry = logfmt.NewEntry().Append(
					logfmt.IP(r.RemoteAddr),
					logfmt.Method(r.Method),
					logfmt.URI(r.URL.String()),
					logfmt.TimeTaken(time.Since(start)),
					logfmt.Status(int(code)),
					logfmt.Bytes(int(bytes)),
				).ToBytes()

				l.Info(string(entry))
			}(r, now)

			h.ServeHTTP(w, r)
		})
	}
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
	var accessPath, errorPath = fmt.Sprintf("/var/log/vroom/access/access-%d", time.Now().Unix()), fmt.Sprintf("/var/log/vroom/error/error-%d", time.Now().Unix())
	var symAccess, symError = "/var/log/vroom/access/sym-access", "/var/log/vroom/error/sym-error"
	if debug {
		accessPath, errorPath = fmt.Sprintf("log/vroom/access/access-%d", time.Now().Unix()), fmt.Sprintf("log/vroom/error/error-%d", time.Now().Unix())
		symAccess, symError = "log/vroom/access/sym-access", "log/vroom/error/sym-error"
	}
	access, erro := initLog(accessPath), initLog(errorPath)
	if access.Size() == 0 {
		access.Write(logfmt.NewDirective("Vroom Proxy", "1.0", "call vroom api", "Date", "Time", "File", "IP", "Method", "URI", "Time Taken", "Status", "Bytes").ToBytes())
	}
	if erro.Size() == 0 {
		erro.Write(logfmt.NewDirective("Vroom Proxy", "1.0", "call vroom api", "Date", "Time", "File", "Error").ToBytes())
	}
	os.Remove(symAccess)
	os.Remove(symError)
	os.Symlink(accessPath, symAccess)
	os.Symlink(errorPath, symError)

	/*
	 * SETUP ROUTER AND MIDDLEWARE FOR LOGGIN REQS
	 */
	r := mux.NewRouter().StrictSlash(true)
	logReq := accessLogger(access)
	r.Methods("GET").Path("/").Name("index").Handler(logReq(indexGet(erro)))
	r.Methods("POST").Path("/").Name("vroomupload").Handler(logReq(indexPost(erro)))
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("page not found"))
	})
	srv.Handler = r

	/*
	 * SETUP GRACEFUL SERVER SHUTDOWN
	 */
	shutdown := make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c)

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

			close(shutdown)
		}
	}(&srv, access, erro)

	fmt.Printf("listening at http://127.0.0.1%s...\n", srv.Addr)
	fmt.Println(srv.ListenAndServe())
	<-shutdown
}