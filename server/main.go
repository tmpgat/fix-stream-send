package main

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/bufbuild/connect-grpcreflect-go"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"darace/pb/pbconnect"
)

type TimeHandler struct {
	chans *sync.Map
}

func NewTimeHandler() *TimeHandler {
	h := &TimeHandler{chans: &sync.Map{}}
	go h.loop()
	return h
}

func (h *TimeHandler) loop() {
	for t := range time.NewTicker(time.Second).C {
		msg := timestamppb.New(t)
		h.chans.Range(func(key, _ any) bool {
			ch := key.(chan *timestamppb.Timestamp)
			go func() {
				// TODO: ensure channel is open before send
				ch <- msg
			}()
			return true
		})
	}
}

func (h *TimeHandler) Now(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[timestamppb.Timestamp], error) {
	return connect.NewResponse(timestamppb.Now()), nil
}

func auxStream[T any](chans *sync.Map, ctx context.Context, _ *connect.ServerStream[T]) chan *T {
	ch := make(chan *T)
	chans.Store(ch, struct{}{})
	go func() {
		<-ctx.Done()
		chans.Delete(ch)
		close(ch)
	}()
	return ch
}

func (h *TimeHandler) Live(ctx context.Context, req *connect.Request[emptypb.Empty], stream *connect.ServerStream[timestamppb.Timestamp]) error {
	_ = stream.Send(nil)
	for msg := range auxStream(h.chans, ctx, stream) {
		_ = stream.Send(msg)
	}
	return nil
}

var _ pbconnect.TimeHandler = &TimeHandler{}

func main() {
	mux := http.NewServeMux()
	mux.Handle(pbconnect.NewTimeHandler(NewTimeHandler()))

	reflector := grpcreflect.NewStaticReflector(pbconnect.TimeName)
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	c := cors.New(cors.Options{
		AllowOriginFunc:  func(string) bool { return true },
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
	h := c.Handler(mux)

	newHandler := h2c.NewHandler(h, &http2.Server{})
	err := http.ListenAndServe(":8484", newHandler)
	if err != nil {
		panic(err)
	}
}
