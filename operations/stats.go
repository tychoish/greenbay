package operations

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/tychoish/gimlet"
	"github.com/tychoish/grip/message"
)

type statsErrorResponse struct {
	Pid   int    `json:"pid,omitempty"`
	Error string `json:"error"`
}

func (s *GreenbayService) sysInfoHandler(w http.ResponseWriter, r *http.Request) {
	info := message.CollectSystemInfo()
	if !info.Loggable() {
		resp := &statsErrorResponse{Error: strings.Join(info.(*message.SystemInfo).Errors, "; ")}
		gimlet.WriteInternalErrorJSON(w, resp)
		return
	}

	gimlet.WriteJSON(w, info)
}

func (s *GreenbayService) processInfoHandler(w http.ResponseWriter, r *http.Request) {
	var pid int32
	pidArg, ok := gimlet.GetVars(r)["pid"]
	if !ok {
		// if no pid is specified (which can happen as this
		// handler is used for a route without a pid), we
		// should just inspect the root pid of the
		// system. Also Pid 0 isn't a thing.
		pid = 1
	} else {
		p, err := strconv.Atoi(pidArg)
		if err != nil {
			gimlet.WriteErrorJSON(w, &statsErrorResponse{
				Error: err.Error(),
			})
			return
		}
		pid = int32(p)
	}

	out := message.CollectProcessInfoWithChildren(int32(pid))

	if len(out) == 0 {
		gimlet.WriteErrorJSON(w, &statsErrorResponse{Pid: int(pid),
			Error: "pid not identified"})
		return
	}

	for _, info := range out {
		if !info.Loggable() {
			resp := &statsErrorResponse{Error: strings.Join(info.(*message.ProcessInfo).Errors, "; ")}
			gimlet.WriteInternalErrorJSON(w, resp)
			return
		}
	}

	gimlet.WriteJSON(w, out)
}
