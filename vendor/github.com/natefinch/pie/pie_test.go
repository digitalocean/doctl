package pie

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

var _ io.ReadWriteCloser = rwCloser{}
var _ io.ReadWriteCloser = ioPipe{}

func TestRWCloser(t *testing.T) {
	rc := &closeRW{}
	wc := &closeRW{}
	rwc := rwCloser{rc, wc}
	if err := rwc.Close(); err != nil {
		t.Errorf("unexpected error from rwCloser.Close: %#v", err)
	}
	if !rc.closed {
		t.Error("Close not called on ReadCloser.")
	}
	if !wc.closed {
		t.Error("Close not called on WriteCloser.")
	}
}

func TestRWCloserReadCloserError(t *testing.T) {
	readCloserErr := errors.New("read")
	rc := &closeRW{err: readCloserErr}
	wc := &closeRW{}
	rwc := rwCloser{rc, wc}
	err := rwc.Close()
	if !rc.closed {
		t.Error("Close not called on ReadCloser.")
	}
	if !wc.closed {
		t.Error("Close not called on WriteCloser.")
	}
	if err == nil {
		t.Error("ReadCloser error not passed through from rwCloser.Close")
	}
	if err != readCloserErr {
		t.Errorf("Different error returned from rwCloser than expected: %#v", err)
	}
}

func TestRWCloserWriteCloserError(t *testing.T) {
	writeCloserErr := errors.New("write")
	rc := &closeRW{}
	wc := &closeRW{err: writeCloserErr}
	rwc := rwCloser{rc, wc}
	err := rwc.Close()
	if !rc.closed {
		t.Error("Close not called on ReadCloser.")
	}
	if !wc.closed {
		t.Error("Close not called on WriteCloser.")
	}
	if err == nil {
		t.Error("ReadCloser error not passed through from rwCloser.Close")
	}
	if err != writeCloserErr {
		t.Errorf("Different error returned from rwCloser than expected: %#v", err)
	}
}

func TestRWCloserBothCloserError(t *testing.T) {
	writeCloserErr := errors.New("write")
	readCloserErr := errors.New("read")
	rc := &closeRW{err: readCloserErr}
	wc := &closeRW{err: writeCloserErr}
	rwc := rwCloser{rc, wc}
	err := rwc.Close()
	if !rc.closed {
		t.Error("Close not called on ReadCloser.")
	}
	if !wc.closed {
		t.Error("Close not called on WriteCloser.")
	}
	if err == nil {
		t.Error("Error not passed through from rwCloser.Close")
	}

	// I don't think we actually care which of these errors gets returned, as
	// long as one of them does.
	if err != writeCloserErr && err != readCloserErr {
		t.Errorf("Different error returned from rwCloser than expected: %#v", err)
	}
}

func TestIOPipeClose(t *testing.T) {
	rc := &closeRW{}
	wc := &closeRW{}
	p := &proc{}
	iop := ioPipe{rc, wc, p}
	if err := iop.Close(); err != nil {
		t.Errorf("Unexpected error from ioPipe.Close: %#v", err)
	}
	if !rc.closed {
		t.Error("Close not called on ReadCloser.")
	}
	if !wc.closed {
		t.Error("Close not called on WriteCloser.")
	}
	if p.sig == nil {
		t.Errorf("No signal sent to process")
	}
	if p.sig != os.Interrupt {
		t.Errorf("Unexpected signal sent to process, expected os.Interrupt, got %#v", p.sig)
	}
	if p.killed {
		t.Errorf("Kill() called unexpectedly on process.")
	}
}

func TestIOPipeSlowProc(t *testing.T) {
	defer func(d time.Duration) {
		procTimeout = d
	}(procTimeout)
	procTimeout = 5 * time.Millisecond
	rc := &closeRW{}
	wc := &closeRW{}
	p := &proc{delay: procTimeout * 2}
	iop := ioPipe{rc, wc, p}
	if err := iop.Close(); err != errProcStopTimeout {
		t.Errorf("Unexpected error from ioPipe.Close, expected %#v, got: %#v", errProcStopTimeout, err)
	}
	if !rc.closed {
		t.Error("Close not called on ReadCloser.")
	}
	if !wc.closed {
		t.Error("Close not called on WriteCloser.")
	}
	if p.sig == nil {
		t.Errorf("no signal sent to process")
	}
	if p.sig != os.Interrupt {
		t.Errorf("Unexpected signal sent to process, expected os.Interrupt, got %#v", p.sig)
	}
	if !p.killed {
		t.Errorf("Kill() unexpectedly not called on process.")
	}
}

func TestNewProvider(t *testing.T) {
	p := NewProvider()
	if p.server == nil {
		t.Error("Unexpected nil rpc Server")
	}
	if p.rwc == nil {
		t.Error("Unexpected nil ReadWriteCloser")
	}
	rwc, ok := p.rwc.(rwCloser)
	if !ok {
		t.Errorf("Expected ReadWriteCloser to be rwCloser, but is %#v", p.rwc)
	}
	if rwc.ReadCloser != os.Stdin {
		t.Errorf("Expected rwc.ReadCloser to be os.Stdin but is %#v", rwc.ReadCloser)
	}
	if rwc.WriteCloser != os.Stdout {
		t.Errorf("Expected rwc.WriteCloser to be os.Stdout but is %#v", rwc.ReadCloser)
	}
}

func TestNewConsumer(t *testing.T) {
	c := NewConsumer()
	if c == nil {
		t.Fatal("Unexpected nil pointer from NewConsumer")
	}
}

func TestNewConsumerCodec(t *testing.T) {
	tcc := &testClientCodec{}
	c := NewConsumerCodec(tcc.NewClientCodec)
	if c == nil {
		t.Fatal("Unexpected nil pointer from NewConsumerCodec")
	}
	if !tcc.called {
		t.Fatal("NewClientCodec function never called.")
	}
}

func TestServeAndStart(t *testing.T) {
	testServeAndStart(nil, nil, t)
}

func TestServeAndStartCodec(t *testing.T) {
	testServeAndStart(jsonrpc.NewServerCodec, jsonrpc.NewClientCodec, t)
}

func testServeAndStart(
	servercodec func(io.ReadWriteCloser) rpc.ServerCodec,
	clientcodec func(io.ReadWriteCloser) rpc.ClientCodec,
	t *testing.T,
) {
	// set up some pipes for reading/writing that we can pretend are
	// stdin and stdout for a plugin application.
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	process := &proc{}

	rwc := rwCloser{
		ReadCloser:  stdinR,
		WriteCloser: stdoutW,
	}

	// now start a plugin provider using these pipes
	s := Server{server: rpc.NewServer(), rwc: rwc}

	api := api{}
	s.RegisterName("api", api)
	api2 := API2{}
	s.Register(api2)

	done := make(chan struct{})

	go func() {
		if servercodec == nil {
			s.Serve()
		} else {
			s.ServeCodec(servercodec)
		}
		close(done)
	}()

	// now we mock out the makeCommand that'll get called by the host.
	f := &fakeCmdData{
		stdout: stdoutR,
		stdin:  stdinW,
		p:      process,
	}
	old := makeCommand
	makeCommand = f.makeCommand
	defer func() { makeCommand = old }()

	output := &bytes.Buffer{}
	path := "foo"
	args := []string{"bar", "baz"}
	var client *rpc.Client
	var err error
	if clientcodec == nil {
		client, err = StartProvider(output, path, args...)
		if err != nil {
			t.Errorf("Unexpected non-nil error from Start: %#v", err)
		}
	} else {
		client, err = StartProviderCodec(clientcodec, output, path, args...)
		if err != nil {
			t.Errorf("Unexpected non-nil error from StartWithCodec: %#v", err)
		}
	}

	if f.w != output {
		t.Error("Output writer not passed to makeCommand")
	}
	if f.path != path {
		t.Error("Path not passed to makeCommand")
	}
	if !reflect.DeepEqual(f.args, args) {
		t.Error("Args not passed to makeCommand")
	}

	name := "bob"
	var response string
	if err := client.Call("api.SayHi", name, &response); err != nil {
		t.Fatalf("Unexpected non-nil error from client.Call: %#v", err)
	}
	var expected string
	api.SayHi(name, &expected)
	if response != expected {
		t.Fatalf("Wrong Response from api call, expected %q, got %q", expected, response)
	}
	if err := client.Call("API2.SayBye", name, &response); err != nil {
		t.Fatalf("Unexpected non-nil error from client.Call: %#v", err)
	}
	api2.SayBye(name, &expected)
	if response != expected {
		t.Fatalf("Wrong Response from API2 call, expected %q, got %q", expected, response)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Unexpected non-nil error from client.Call: %#v", err)
	}
	select {
	case <-done:
		// pass
	case <-time.After(time.Millisecond * 10):
		t.Fatal("Server failed to stop after close in 10ms")
	}
}

func TestConsumerClientClose(t *testing.T) {
	testConsumer(false, nil, nil, t)
}

func TestConsumerServerClose(t *testing.T) {
	testConsumer(true, nil, nil, t)
}

func TestConsumerCodecClientCLose(t *testing.T) {
	testConsumer(false, jsonrpc.NewServerCodec, jsonrpc.NewClientCodec, t)
}

func TestConsumerCodecServerClose(t *testing.T) {
	testConsumer(true, jsonrpc.NewServerCodec, jsonrpc.NewClientCodec, t)
}

func testConsumer(
	closeServer bool,
	servercodec func(io.ReadWriteCloser) rpc.ServerCodec,
	clientcodec func(io.ReadWriteCloser) rpc.ClientCodec,
	t *testing.T,
) {
	// set up some pipes for reading/writing that we can pretend are
	// stdin and stdout for a plugin application.
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	process := &proc{}

	// mock out the makeCommand that'll get called by the host.
	f := &fakeCmdData{
		stdout: stdoutR,
		stdin:  stdinW,
		p:      process,
	}
	old := makeCommand
	makeCommand = f.makeCommand
	defer func() { makeCommand = old }()
	output := &bytes.Buffer{}

	path := "foo"
	args := []string{"bar", "baz"}
	server, err := StartConsumer(output, "foo", args...)
	if err != nil {
		t.Fatalf("Unexpected error from StartConsumer: %#v", err)
	}

	if f.w != output {
		t.Error("Output writer not passed to makeCommand")
	}
	if f.path != path {
		t.Error("Path not passed to makeCommand")
	}
	if !reflect.DeepEqual(f.args, args) {
		t.Error("Args not passed to makeCommand")
	}

	api := api{}
	server.RegisterName("api", api)
	api2 := API2{}
	server.Register(api2)

	done := make(chan struct{})

	go func() {
		if servercodec == nil {
			server.Serve()
		} else {
			server.ServeCodec(servercodec)
		}
		close(done)
	}()

	var client *rpc.Client
	if clientcodec == nil {
		client = rpc.NewClient(rwCloser{stdinR, stdoutW})
	} else {
		client = rpc.NewClientWithCodec(clientcodec(rwCloser{stdinR, stdoutW}))
	}
	defer client.Close()

	name := "bob"
	var response string
	if err := client.Call("api.SayHi", name, &response); err != nil {
		t.Fatalf("Unexpected non-nil error from client.Call: %#v", err)
	}
	var expected string
	api.SayHi(name, &expected)
	if response != expected {
		t.Fatalf("Wrong Response from api call, expected %q, got %q", expected, response)
	}

	if err := client.Call("API2.SayBye", name, &response); err != nil {
		t.Fatalf("Unexpected non-nil error from client.Call: %#v", err)
	}
	api2.SayBye(name, &expected)
	if response != expected {
		t.Fatalf("Wrong Response from api2 call, expected %q, got %q", expected, response)
	}

	if closeServer {
		if err := server.Close(); err != nil {
			t.Fatalf("Unexpected non-nil error from server.Close: %#v", err)
		}
	} else {
		if err := client.Close(); err != nil {
			t.Fatalf("Unexpected non-nil error from client.Call: %#v", err)
		}
	}
	select {
	case <-done:
		// pass
	case <-time.After(time.Millisecond * 10):
		t.Fatal("Server failed to stop after close in 10ms")
	}

	if closeServer && !process.waited {
		t.Fatal("Server was closed, but process was not stopped")
	}
}

func TestMakeCommandAndStart(t *testing.T) {
	path := "echo"
	args := []string{"something"}
	c := makeCommand(nil, path, args)
	_, ok := c.(execCmd)
	if !ok {
		t.Fatalf("Expected commander to be type execCmd, but was %#v", c)
	}
	pipe, err := start(c)
	if err != nil {
		t.Fatalf("Unexpected non-nil error: %#v", err)
	}
	defer pipe.Close()
	if pipe.proc == nil {
		t.Fatal("Unexpected nil proc in ioPipe")
	}
	p, ok := pipe.proc.(*os.Process)
	if !ok {
		t.Fatalf("Expected proc to be os.Process but was %#v", pipe.proc)
	}
	var out []byte
	var readerr error
	readFinished := make(chan struct{})
	go func() {
		out, readerr = ioutil.ReadAll(pipe.ReadCloser)
		close(readFinished)
	}()
	go func() {
		// When the process finishes, the above ioutil.ReadAll will complete as
		// well.
		p.Wait()
	}()
	select {
	case <-time.After(time.Millisecond * 100):
		p.Kill()
		t.Fatalf("Timed out waiting for process to run")
	case <-readFinished:
		if readerr != nil {
			t.Fatalf("Unexpected error reading from the process' stdout: %#v", readerr)
		}
		actual := strings.TrimSpace(string(out))
		if actual != args[0] {
			t.Fatalf("Wrong output, expected %q, got %q", args[0], actual)
		}

	}
}

type testClientCodec struct {
	called bool
}

func (t *testClientCodec) NewClientCodec(r io.ReadWriteCloser) rpc.ClientCodec {
	t.called = true
	return jsonrpc.NewClientCodec(r)
}

func fakeServerCodec(conn io.ReadWriteCloser) rpc.ServerCodec {
	return nil
}

type fakeCmdData struct {
	stdout io.ReadCloser
	stdin  io.WriteCloser
	p      *proc
	w      io.Writer
	path   string
	args   []string
}

func (f *fakeCmdData) makeCommand(w io.Writer, path string, args []string) commander {
	f.w = w
	f.path = path
	f.args = args
	return fakeCommand{f.stdin, f.stdout, f.p}
}

type nopWCloser struct {
	io.Writer
}

func (nopWCloser) Close() error { return nil }

type fakeCommand struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
	p      *proc
}

func (f fakeCommand) Start() (osProcess, error) {
	return f.p, nil
}

func (f fakeCommand) StdinPipe() (io.WriteCloser, error) {
	return f.stdin, nil
}

func (f fakeCommand) StdoutPipe() (io.ReadCloser, error) {
	return f.stdout, nil
}

type api struct{}

func (api) SayHi(name string, response *string) error {
	*response = "Hi " + name
	return nil
}

type API2 struct{}

func (API2) SayBye(name string, response *string) error {
	*response = "Bye " + name
	return nil
}

// proc is a helper that fullfills the osProcess interface for testing purposes.
type proc struct {
	mu        sync.Mutex
	delay     time.Duration
	waitErr   error
	killErr   error
	signalErr error
	sig       os.Signal
	killed    bool
	waited    bool
}

// Wait will wait for delay time and then return waitErr.
func (p *proc) Wait() (*os.ProcessState, error) {
	p.mu.Lock()
	p.waited = true
	p.mu.Unlock()
	<-time.After(p.delay)
	return nil, p.waitErr
}

// Kill returns killErr.
func (p *proc) Kill() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.killed = true
	return p.killErr
}

// Signal ignores the signal and returns signalErr.
func (p *proc) Signal(sig os.Signal) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sig = sig
	return p.signalErr
}

// closeRW is a helper that fulfills io.Reader, io.Writer, and io.Closer for
// testing purposes.
type closeRW struct {
	closed bool
	err    error
}

// Close fulfills io.Closer and will record that it was called, and return this
// value's error, if any.
func (c *closeRW) Close() error {
	c.closed = true
	return c.err
}

// Read fulfills io.Reader and does nothing.
func (*closeRW) Read(_ []byte) (int, error) {
	return 0, nil
}

// Write fulfills io.Writer and does nothing.
func (*closeRW) Write(_ []byte) (int, error) {
	return 0, nil
}
