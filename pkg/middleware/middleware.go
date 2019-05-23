package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/random9s/cinder/logger"
	logfmt "github.com/random9s/cinder/logger/format"
)

func AccessLogger(l logger.Logger) func(http.Handler) http.Handler {
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
