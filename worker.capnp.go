// Code generated by capnpc-go. DO NOT EDIT.

package worker

import (
	capnp "capnproto.org/go/capnp/v3"
	text "capnproto.org/go/capnp/v3/encoding/text"
	fc "capnproto.org/go/capnp/v3/flowcontrol"
	schemas "capnproto.org/go/capnp/v3/schemas"
	server "capnproto.org/go/capnp/v3/server"
	context "context"
)

type Worker capnp.Client

// Worker_TypeID is the unique identifier for the type Worker.
const Worker_TypeID = 0xc4ef214b3fa3e0f9

func (c Worker) Assign(ctx context.Context, params func(Worker_assign_Params) error) (Worker_assign_Results_Future, capnp.ReleaseFunc) {

	s := capnp.Send{
		Method: capnp.Method{
			InterfaceID:   0xc4ef214b3fa3e0f9,
			MethodID:      0,
			InterfaceName: "worker.capnp:Worker",
			MethodName:    "assign",
		},
	}
	if params != nil {
		s.ArgsSize = capnp.ObjectSize{DataSize: 0, PointerCount: 1}
		s.PlaceArgs = func(s capnp.Struct) error { return params(Worker_assign_Params(s)) }
	}

	ans, release := capnp.Client(c).SendCall(ctx, s)
	return Worker_assign_Results_Future{Future: ans.Future()}, release

}

func (c Worker) WaitStreaming() error {
	return capnp.Client(c).WaitStreaming()
}

// String returns a string that identifies this capability for debugging
// purposes.  Its format should not be depended on: in particular, it
// should not be used to compare clients.  Use IsSame to compare clients
// for equality.
func (c Worker) String() string {
	return "Worker(" + capnp.Client(c).String() + ")"
}

// AddRef creates a new Client that refers to the same capability as c.
// If c is nil or has resolved to null, then AddRef returns nil.
func (c Worker) AddRef() Worker {
	return Worker(capnp.Client(c).AddRef())
}

// Release releases a capability reference.  If this is the last
// reference to the capability, then the underlying resources associated
// with the capability will be released.
//
// Release will panic if c has already been released, but not if c is
// nil or resolved to null.
func (c Worker) Release() {
	capnp.Client(c).Release()
}

// Resolve blocks until the capability is fully resolved or the Context
// expires.
func (c Worker) Resolve(ctx context.Context) error {
	return capnp.Client(c).Resolve(ctx)
}

func (c Worker) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Client(c).EncodeAsPtr(seg)
}

func (Worker) DecodeFromPtr(p capnp.Ptr) Worker {
	return Worker(capnp.Client{}.DecodeFromPtr(p))
}

// IsValid reports whether c is a valid reference to a capability.
// A reference is invalid if it is nil, has resolved to null, or has
// been released.
func (c Worker) IsValid() bool {
	return capnp.Client(c).IsValid()
}

// IsSame reports whether c and other refer to a capability created by the
// same call to NewClient.  This can return false negatives if c or other
// are not fully resolved: use Resolve if this is an issue.  If either
// c or other are released, then IsSame panics.
func (c Worker) IsSame(other Worker) bool {
	return capnp.Client(c).IsSame(capnp.Client(other))
}

// Update the flowcontrol.FlowLimiter used to manage flow control for
// this client. This affects all future calls, but not calls already
// waiting to send. Passing nil sets the value to flowcontrol.NopLimiter,
// which is also the default.
func (c Worker) SetFlowLimiter(lim fc.FlowLimiter) {
	capnp.Client(c).SetFlowLimiter(lim)
}

// Get the current flowcontrol.FlowLimiter used to manage flow control
// for this client.
func (c Worker) GetFlowLimiter() fc.FlowLimiter {
	return capnp.Client(c).GetFlowLimiter()
}

// A Worker_Server is a Worker with a local implementation.
type Worker_Server interface {
	Assign(context.Context, Worker_assign) error
}

// Worker_NewServer creates a new Server from an implementation of Worker_Server.
func Worker_NewServer(s Worker_Server) *server.Server {
	c, _ := s.(server.Shutdowner)
	return server.New(Worker_Methods(nil, s), s, c)
}

// Worker_ServerToClient creates a new Client from an implementation of Worker_Server.
// The caller is responsible for calling Release on the returned Client.
func Worker_ServerToClient(s Worker_Server) Worker {
	return Worker(capnp.NewClient(Worker_NewServer(s)))
}

// Worker_Methods appends Methods to a slice that invoke the methods on s.
// This can be used to create a more complicated Server.
func Worker_Methods(methods []server.Method, s Worker_Server) []server.Method {
	if cap(methods) == 0 {
		methods = make([]server.Method, 0, 1)
	}

	methods = append(methods, server.Method{
		Method: capnp.Method{
			InterfaceID:   0xc4ef214b3fa3e0f9,
			MethodID:      0,
			InterfaceName: "worker.capnp:Worker",
			MethodName:    "assign",
		},
		Impl: func(ctx context.Context, call *server.Call) error {
			return s.Assign(ctx, Worker_assign{call})
		},
	})

	return methods
}

// Worker_assign holds the state for a server call to Worker.assign.
// See server.Call for documentation.
type Worker_assign struct {
	*server.Call
}

// Args returns the call's arguments.
func (c Worker_assign) Args() Worker_assign_Params {
	return Worker_assign_Params(c.Call.Args())
}

// AllocResults allocates the results struct.
func (c Worker_assign) AllocResults() (Worker_assign_Results, error) {
	r, err := c.Call.AllocResults(capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Worker_assign_Results(r), err
}

// Worker_List is a list of Worker.
type Worker_List = capnp.CapList[Worker]

// NewWorker creates a new list of Worker.
func NewWorker_List(s *capnp.Segment, sz int32) (Worker_List, error) {
	l, err := capnp.NewPointerList(s, sz)
	return capnp.CapList[Worker](l), err
}

type Worker_assign_Params capnp.Struct

// Worker_assign_Params_TypeID is the unique identifier for the type Worker_assign_Params.
const Worker_assign_Params_TypeID = 0x8612f7506f622fbd

func NewWorker_assign_Params(s *capnp.Segment) (Worker_assign_Params, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Worker_assign_Params(st), err
}

func NewRootWorker_assign_Params(s *capnp.Segment) (Worker_assign_Params, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Worker_assign_Params(st), err
}

func ReadRootWorker_assign_Params(msg *capnp.Message) (Worker_assign_Params, error) {
	root, err := msg.Root()
	return Worker_assign_Params(root.Struct()), err
}

func (s Worker_assign_Params) String() string {
	str, _ := text.Marshal(0x8612f7506f622fbd, capnp.Struct(s))
	return str
}

func (s Worker_assign_Params) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Struct(s).EncodeAsPtr(seg)
}

func (Worker_assign_Params) DecodeFromPtr(p capnp.Ptr) Worker_assign_Params {
	return Worker_assign_Params(capnp.Struct{}.DecodeFromPtr(p))
}

func (s Worker_assign_Params) ToPtr() capnp.Ptr {
	return capnp.Struct(s).ToPtr()
}
func (s Worker_assign_Params) IsValid() bool {
	return capnp.Struct(s).IsValid()
}

func (s Worker_assign_Params) Message() *capnp.Message {
	return capnp.Struct(s).Message()
}

func (s Worker_assign_Params) Segment() *capnp.Segment {
	return capnp.Struct(s).Segment()
}
func (s Worker_assign_Params) Wasm() ([]byte, error) {
	p, err := capnp.Struct(s).Ptr(0)
	return []byte(p.Data()), err
}

func (s Worker_assign_Params) HasWasm() bool {
	return capnp.Struct(s).HasPtr(0)
}

func (s Worker_assign_Params) SetWasm(v []byte) error {
	return capnp.Struct(s).SetData(0, v)
}

// Worker_assign_Params_List is a list of Worker_assign_Params.
type Worker_assign_Params_List = capnp.StructList[Worker_assign_Params]

// NewWorker_assign_Params creates a new list of Worker_assign_Params.
func NewWorker_assign_Params_List(s *capnp.Segment, sz int32) (Worker_assign_Params_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1}, sz)
	return capnp.StructList[Worker_assign_Params](l), err
}

// Worker_assign_Params_Future is a wrapper for a Worker_assign_Params promised by a client call.
type Worker_assign_Params_Future struct{ *capnp.Future }

func (f Worker_assign_Params_Future) Struct() (Worker_assign_Params, error) {
	p, err := f.Future.Ptr()
	return Worker_assign_Params(p.Struct()), err
}

type Worker_assign_Results capnp.Struct

// Worker_assign_Results_TypeID is the unique identifier for the type Worker_assign_Results.
const Worker_assign_Results_TypeID = 0xb2ad9e8e7b416561

func NewWorker_assign_Results(s *capnp.Segment) (Worker_assign_Results, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Worker_assign_Results(st), err
}

func NewRootWorker_assign_Results(s *capnp.Segment) (Worker_assign_Results, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Worker_assign_Results(st), err
}

func ReadRootWorker_assign_Results(msg *capnp.Message) (Worker_assign_Results, error) {
	root, err := msg.Root()
	return Worker_assign_Results(root.Struct()), err
}

func (s Worker_assign_Results) String() string {
	str, _ := text.Marshal(0xb2ad9e8e7b416561, capnp.Struct(s))
	return str
}

func (s Worker_assign_Results) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Struct(s).EncodeAsPtr(seg)
}

func (Worker_assign_Results) DecodeFromPtr(p capnp.Ptr) Worker_assign_Results {
	return Worker_assign_Results(capnp.Struct{}.DecodeFromPtr(p))
}

func (s Worker_assign_Results) ToPtr() capnp.Ptr {
	return capnp.Struct(s).ToPtr()
}
func (s Worker_assign_Results) IsValid() bool {
	return capnp.Struct(s).IsValid()
}

func (s Worker_assign_Results) Message() *capnp.Message {
	return capnp.Struct(s).Message()
}

func (s Worker_assign_Results) Segment() *capnp.Segment {
	return capnp.Struct(s).Segment()
}
func (s Worker_assign_Results) Result() (string, error) {
	p, err := capnp.Struct(s).Ptr(0)
	return p.Text(), err
}

func (s Worker_assign_Results) HasResult() bool {
	return capnp.Struct(s).HasPtr(0)
}

func (s Worker_assign_Results) ResultBytes() ([]byte, error) {
	p, err := capnp.Struct(s).Ptr(0)
	return p.TextBytes(), err
}

func (s Worker_assign_Results) SetResult(v string) error {
	return capnp.Struct(s).SetText(0, v)
}

// Worker_assign_Results_List is a list of Worker_assign_Results.
type Worker_assign_Results_List = capnp.StructList[Worker_assign_Results]

// NewWorker_assign_Results creates a new list of Worker_assign_Results.
func NewWorker_assign_Results_List(s *capnp.Segment, sz int32) (Worker_assign_Results_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1}, sz)
	return capnp.StructList[Worker_assign_Results](l), err
}

// Worker_assign_Results_Future is a wrapper for a Worker_assign_Results promised by a client call.
type Worker_assign_Results_Future struct{ *capnp.Future }

func (f Worker_assign_Results_Future) Struct() (Worker_assign_Results, error) {
	p, err := f.Future.Ptr()
	return Worker_assign_Results(p.Struct()), err
}

const schema_95c0f3edf8dd266c = "x\xda\x12\xe8u`1\xe4\xcdgb`\x0a\x94ae" +
	"\xfb\xbfW?)?\xe0\xbbP\x1b\x83\xa0\x08#\x03\x03" +
	"+#;\x03\x83\xb1,\xa3\x10#\x03\xa3\xb0*\xa3=" +
	"\x03\xe3\xff\xc4T\xc7\xea\xbeyk7!+pe\x94" +
	"\x02)\xf0\x05+\xf8\xf9`\xb1\xbd\xb7\xe2\xfb#\x0c\x82" +
	"\xbc\xcc\xffs\xd4\xee\xfex\xfb\xf9\xc0T\x06\x06F\xe1" +
	"\\\xc6E\xc2\xa5 \xf5\xc2\x85\x8c\xee\xc23\x19\xd9\x19" +
	"t\xfe\x97\xe7\x17e\xa7\x16\xe9%3'\x16\xe4\x15X" +
	"\x85Cx\x89\xc5\xc5\x99\xe9y*\x01\x89E\x89\xb9\x8c" +
	"\xc5\x81,\xcc,\x0c\x0c,\x8c\x0c\x0c\x82\xbcZ\x0c\x0c" +
	"\x81\x1c\xcc\x8c\x81\"L\x8c\xfc\xe5\x89\xc5\xb9\x8c\xbc\x0c" +
	"L\x8c\xbc\x0c\x8cx\xcd\x09J-.\xcda.A1" +
	"\xc8\x0aa\x90}\x11H\xbe\x84\x91\x87\x81\x89\x91\x07\xc9" +
	"(F\x98Q\xec\xd9\xa9E\x01\x8c\x8c\x81,\xcc\xac\x0c" +
	"\x0c\xf0\x00b\x84\x05\x84\xa0\xa0\x15\x03\x93 +\xbb=" +
	"\xc4:\x07\xc6\x00FF@\x00\x00\x00\xff\xff\x00zW" +
	"\xd5"

func RegisterSchema(reg *schemas.Registry) {
	reg.Register(&schemas.Schema{
		String: schema_95c0f3edf8dd266c,
		Nodes: []uint64{
			0x8612f7506f622fbd,
			0xb2ad9e8e7b416561,
			0xc4ef214b3fa3e0f9,
		},
		Compressed: true,
	})
}
