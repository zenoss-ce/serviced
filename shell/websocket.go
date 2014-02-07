package shell

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/googollee/go-socket.io"
	"github.com/gorilla/websocket"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"
)

const (
	FORK   = "FORK"
	EXEC   = "EXEC"
	SIGNAL = "SIGNAL"
)

// Client->Agent protocol
type request struct {
	Action    string
	ServiceId string
	Env       []string
	Cmd       string
	Signal    int
}

// Agent->Client protocol
type response struct {
	Stdin  string
	Stdout string
	Stderr string
	Result string
}

// Defines commands to be run in an object's container
type Process struct {
	ServiceId string              // The service id of the container to start
	IsTTY     bool                // Describes the type of connection needed
	Envv      []string            // Environment variables
	Command   string              // Command to run
	Error     error               `json:"-"`
	Stdin     chan string         `json:"-"`
	Stdout    chan string         `json:"-"`
	Stderr    chan string         `json:"-"`
	Exited    chan bool           `json:"-"`
	Signal    chan syscall.Signal `json:"-"`
	whenDone  chan bool
	done      bool
}

func NewProcess(serviceId, command string, envv []string, istty bool) *Process {
	return &Process{
		ServiceId: serviceId,
		IsTTY:     istty,
		Envv:      envv,
		Command:   command,
		Stdin:     make(chan string),
		Stdout:    make(chan string),
		Stderr:    make(chan string),
		Signal:    make(chan syscall.Signal),
		Exited:    make(chan bool),
		whenDone:  make(chan bool),
		done:      false,
	}
}

// Starts a container shell
func Exec(p *Process, s *dao.Service) error {
	var runner Runner

	// Bind mount on /serviced
	dir, bin, err := serviced.ExecPath()
	if err != nil {
		return err
	}
	servicedVolume := fmt.Sprintf("%s:/serviced", dir)

	// Bind mount the pwd
	dir, err = os.Getwd()
	pwdVolume := fmt.Sprintf("%s:/mnt/pwd", dir)

	// Get the shell command
	var shellCmd string
	if p.Command != "" {
		shellCmd = p.Command
	} else {
		shellCmd = "su -"
	}

	// Get the proxy Command
	proxyCmd := []string{fmt.Sprintf("/serviced/%s", bin), "-logtostderr=false", "proxy", "-logstash=false", "-autorestart=false", s.Id, shellCmd}
	// Get the docker start command
	docker, err := exec.LookPath("docker")
	if err != nil {
		return err
	}
	argv := []string{"run", "-rm", "-v", servicedVolume, "-v", pwdVolume}
	argv = append(argv, p.Envv...)

	if p.IsTTY {
		argv = append(argv, "-i", "-t")
	}

	argv = append(argv, s.ImageId)
	argv = append(argv, proxyCmd...)

	runner, err = CreateCommand(docker, argv)

	if err != nil {
		return err
	}

	// @see http://dave.cheney.net/tag/golang-3
	p.Stdout = runner.StdoutPipe()
	p.Stderr = runner.StderrPipe()

	go p.send(runner)
	return nil
}

func (p *Process) send(r Runner) {
	go r.Reader(8192)
	go func() {
		for {
			select {
			case m := <-p.Stdin:
				r.Write([]byte(m))
			case s := <-p.Signal:
				r.Signal(s)
			}
		}
	}()

	<-r.ExitedPipe()
	if e := r.Error(); e != nil {
		p.Error = e
	} else {
		p.Error = errors.New("0")
	}
	p.Exited <- true
	p.whenDone <- true
}

func (p *Process) Wait() {
	if !p.done {
		p.done = <-p.whenDone
	}
}

// Describes streams from an agent-executed process to a client
type ProcessStream interface {

	// Initiate client-side communication and create Process
	StreamClient(http.ResponseWriter, *http.Request, chan *Process)

	// Initiate agent-side communication and kick off shell
	StreamAgent()

	// Wait for the process to end
	Wait()
}

type baseProcessStream struct {
	agent   *websocket.Conn
	process *Process
	addr    string
}

type WebsocketProcessStream struct {
	*baseProcessStream
	client *websocket.Conn
}

type HTTPProcessStream struct {
	*baseProcessStream
	client *net.Conn
}

func NewWebsocketProcessStream(addr string) *WebsocketProcessStream {
	return &WebsocketProcessStream{
		baseProcessStream: &baseProcessStream{addr: addr},
	}
}

func NewHTTPProcessStream(addr string) *HTTPProcessStream {
	return &HTTPProcessStream{
		baseProcessStream: &baseProcessStream{addr: addr},
	}
}

type WebsocketProcessHandler struct {
	Addr string
}

type OSProcessHandler struct {
	Port string
}

type HTTPProcessHandler struct {
	Addr string
}

type SocketIOProcessHandler struct {
	server *socketio.SocketIOServer
}

// Implement http.Handler
func (h *WebsocketProcessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	stream := NewWebsocketProcessStream(h.Addr)

	// Create a client and wait for the process packet
	pc := make(chan bool)

	// Set up everything to start the connection to agent once a process is
	// defined.
	go func() {
		<-pc
		// Now that we have the process, connect to the agent
		stream.StreamAgent()
	}()

	// Now start pulling from the client until we receive a process, then
	// hook it all up
	go stream.StreamClient(w, r, pc)

	// Wait for the process to die
	stream.Wait()
}

func (h *HTTPProcessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// First, get the Process from the POST body, failing if invalid
	data, _ := ioutil.ReadAll(r.Body)
	req := &request{}
	err := json.Unmarshal(data, req)
	if err != nil {
		http.Error(w, "Invalid process descriptor", http.StatusBadRequest)
		return
	}
	proc := reqToProcess(req)
	if proc == nil {
		http.Error(w, "Invalid process descriptor", http.StatusBadRequest)
		return
	}

	// Create the stream
	stream := NewHTTPProcessStream(h.Addr)
	stream.process = proc

	// Start up the agent stream
	go stream.StreamAgent()

	// Start forwarding things to the HTTP client
	go stream.StreamClient(w, r)

	// Wait for the process to die
	stream.Wait()
}

func reqToProcess(req *request) *Process {
	var istty bool
	switch req.Action {
	case FORK:
		istty = true
	case EXEC:
		istty = false
	default:
		return nil
	}
	proc := NewProcess(req.ServiceId, req.Cmd, req.Env, istty)
	if proc.Envv == nil {
		proc.Envv = []string{}
	}
	return proc

}

// Read the first packet from the client and deserialize to Process
func readProcessPacket(ws *websocket.Conn) *Process {
	req := &request{}
	if err := ws.ReadJSON(req); err != nil {
		return nil
	}
	return reqToProcess(req)
}

func (s *WebsocketProcessStream) StreamClient(w http.ResponseWriter, r *http.Request, pc chan bool) {
	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		return
	}
	s.client = ws
	s.process = readProcessPacket(ws)
	pc <- true
	forwardToClient(s.client, s.process)
}

func (s *HTTPProcessStream) StreamClient(w http.ResponseWriter, r *http.Request) {
	// Writer
	for {
		select {
		case m := <-s.process.Stdout:
			fmt.Fprint(w, m)
			w.(http.Flusher).Flush()
		case m := <-s.process.Stderr:
			fmt.Fprint(w, m)
			w.(http.Flusher).Flush()
		case <-s.process.Exited:
			fmt.Fprintf(w, "\nProcess exited: %q\n", s.process.Error)
			w.(http.Flusher).Flush()
			return
		}
	}
}

func (s *baseProcessStream) StreamAgent() {
	// TODO: Proper ws scheme validation
	ws, _, _ := websocket.DefaultDialer.Dial("ws://"+s.addr, nil)
	s.agent = ws

	action := "EXEC"
	if s.process.IsTTY {
		action = "FORK"
	}

	// Recreate the request from the process and send it up the pipe
	req := request{
		Cmd:       s.process.Command,
		Action:    action,
		ServiceId: s.process.ServiceId,
		Env:       s.process.Envv,
	}
	s.agent.WriteJSON(req)

	s.forwardFromAgent()
}

func (s *baseProcessStream) Wait() {
	for {
		if s.process != nil {
			s.process.Wait()
			return
		}
		time.Sleep(10)
	}
}

// Wire up the Process to the agent connection
func (s *baseProcessStream) forwardFromAgent() {
	defer func() {
		s.agent.Close()
		if s.process.Error == nil {
			s.process.Error = errors.New("Connection closed unexpectedly")
			s.process.Exited <- true
		}
	}()

	// Writer
	go func() {
		for {
			select {
			case m := <-s.process.Stdin:
				s.agent.WriteJSON(request{Action: EXEC, Cmd: m})
			case m := <-s.process.Signal:
				s.agent.WriteJSON(request{Action: SIGNAL, Signal: int(m)})
			}
		}
	}()

	// Reader
	for {
		var res response
		if err := s.agent.ReadJSON(&res); err == io.EOF {
			break
		} else if err != nil {
			// Bad read send message
		}

		if res.Stdout != "" {
			s.process.Stdout <- res.Stdout
		}

		if res.Stderr != "" {
			s.process.Stderr <- res.Stderr
		}

		if res.Result != "" {
			s.process.Error = errors.New(res.Result)
			s.process.Exited <- true
			break
		}
	}
}

// Wire up the Process to the client connection
func forwardToClient(ws *websocket.Conn, proc *Process) {
	defer func() {
		ws.Close()
		proc.Signal <- syscall.SIGKILL // Does nothing if process exited
	}()

	// Reader
	go func() {
		for {
			var req request
			if err := ws.ReadJSON(&req); err == io.EOF {
				break
			} else if err != nil {
				// Bad read send message
			}

			switch req.Action {
			// TODO: Defend against invalid requests?
			case SIGNAL:
				proc.Signal <- syscall.Signal(req.Signal)
			case EXEC:
				proc.Stdin <- req.Cmd
			}
		}
	}()

	// Writer
	for {
		select {
		case m := <-proc.Stdout:
			ws.WriteJSON(response{Stdout: m})
		case m := <-proc.Stderr:
			ws.WriteJSON(response{Stderr: m})
		case <-proc.Exited:
			ws.WriteJSON(response{Result: fmt.Sprint(proc.Error)})
			break
		}
	}

}

// This is the handler on the agent that receives the connection from the proxy
func (h *OSProcessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		service  *dao.Service
		empty    interface{}
		services []*dao.Service
	)
	// Establish the websocket connection with proxy
	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		return
	}
	defer ws.Close()

	// Read the process off the websocket
	proc := readProcessPacket(ws)

	// Make it go
	controlplane, err := serviced.NewControlClient(h.Port)
	if err != nil {
		glog.Fatalf("Could not create a control plane client %v", err)
	}

	err = (*controlplane).GetServices(&empty, &services)
	for _, svc := range services {
		if svc.Id == proc.ServiceId || svc.Name == proc.ServiceId {
			service = svc
			break
		}
	}

	if err := Exec(proc, service); err != nil {
		// TODO: Something errory
	}

	// Wire it up
	go forwardToClient(ws, proc)

	proc.Wait()

}
