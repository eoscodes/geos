package asio

import (
		"io"
	"net"
)

type ReactiveSocket struct {
	ctx *IoContext
}

func NewReactiveSocket(ctx *IoContext) *ReactiveSocket {
	d := new(ReactiveSocket)
	d.ctx = ctx
	return d
}

func (r *ReactiveSocket) AsyncAccept(listen net.Listener, op func(conn net.Conn, err error)) {
	// call net.Listener.Accept to block goroutine, new routine will exit after accept a connection
	// callback operation will be executed in io_service in the correct time
	// use ec.Error to get error from accepting event when ec.Valid is true
	go r.accept(listen, op)
}

func (r *ReactiveSocket) accept(listen net.Listener, op func(conn net.Conn, err error)) {
	connect, err := listen.Accept()
	r.ctx.GetService().post(socketAcceptOp{op, connect, err})
}

func (r *ReactiveSocket) AsyncRead(reader io.Reader, b []byte, op func(n int, err error)) {
	// call io.Reader.Read to block goroutine, new routine will exit after reading event
	// callback operation will be executed in io_service in the correct time
	// use ec.Error to get error from reading event when ec.Valid is true
	go r.read(reader, b, op)
}

func (r *ReactiveSocket) read(reader io.Reader, b []byte, op func(n int, err error)) {
	n, err := reader.Read(b)
	r.ctx.GetService().post(socketReadOp{op, n, err})
}

func (r *ReactiveSocket) AsyncReadFull(reader io.Reader, b []byte, op func(n int, err error)) {
	// call io.ReadFull(io.Reader, []byte) to block goroutine, new routine will exit after reading event
	// callback operation will be executed in io_service in the correct time
	// use ec.Error to get error from reading event when ec.Valid is true
	go r.readFull(reader, b, op)
}

func (r *ReactiveSocket) readFull(reader io.Reader, b []byte, op func(n int, err error)) {
	n, err := io.ReadFull(reader, b)
	r.ctx.GetService().post(socketReadFullOp{op, n, err})
}


func (r *ReactiveSocket) AsyncWrite(writer io.Writer, b []byte, op func(n int, ec error)) {
	// call io.Writer.Write to block goroutine, new routine will exit after writing event
	// callback operation will be executed in io_service in the correct time
	// use ec.Error to get error from writing event when ec.Valid is true
	go r.write(writer, b, op)
}

func (r *ReactiveSocket) write(writer io.Writer, b []byte, op func(n int, ec error)) {
	n, err := writer.Write(b)
	r.ctx.GetService().post(socketWriteOp{op, n, err})
}





