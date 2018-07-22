package atreugo

import (
	"bufio"
	"errors"
	"testing"
	"time"

	"github.com/erikdubbelboer/fasthttp"
	"github.com/erikdubbelboer/fasthttp/fasthttputil"
)

func TestAtreugoServer(t *testing.T) {
	type args struct {
		viewFn        View
		middlewareFns []Middleware
	}
	type want struct {
		statusCode        int
		viewCalled        bool
		middleWareCounter int
	}

	viewCalled := false
	middleWareCounter := 0

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "AllOk",
			args: args{
				viewFn: func(ctx *fasthttp.RequestCtx) error {
					viewCalled = true
					return nil
				},
				middlewareFns: []Middleware{
					func(ctx *fasthttp.RequestCtx) (int, error) {
						middleWareCounter++
						return 0, nil
					},
				},
			},
			want: want{
				statusCode:        fasthttp.StatusOK,
				viewCalled:        true,
				middleWareCounter: 1,
			},
		},
		{
			name: "FirstMiddlewareError",
			args: args{
				viewFn: func(ctx *fasthttp.RequestCtx) error {
					viewCalled = true
					return nil
				},
				middlewareFns: []Middleware{
					func(ctx *fasthttp.RequestCtx) (int, error) {
						return 403, errors.New("Bad request")
					},
					func(ctx *fasthttp.RequestCtx) (int, error) {
						middleWareCounter++
						return 0, nil
					},
				},
			},
			want: want{
				statusCode:        403,
				viewCalled:        false,
				middleWareCounter: 0,
			},
		},
		{
			name: "SecondMiddlewareError",
			args: args{
				viewFn: func(ctx *fasthttp.RequestCtx) error {
					viewCalled = true
					return nil
				},
				middlewareFns: []Middleware{
					func(ctx *fasthttp.RequestCtx) (int, error) {
						middleWareCounter++
						return 0, nil
					},
					func(ctx *fasthttp.RequestCtx) (int, error) {
						return 403, errors.New("Bad request")
					},
				},
			},
			want: want{
				statusCode:        403,
				viewCalled:        false,
				middleWareCounter: 1,
			},
		},
		{
			name: "ViewError",
			args: args{
				viewFn: func(ctx *fasthttp.RequestCtx) error {
					viewCalled = true
					return errors.New("Fake error")
				},
				middlewareFns: []Middleware{
					func(ctx *fasthttp.RequestCtx) (int, error) {
						middleWareCounter++
						return 0, nil
					},
				},
			},
			want: want{
				statusCode:        fasthttp.StatusInternalServerError,
				viewCalled:        true,
				middleWareCounter: 1,
			},
		},
	}

	for _, tt := range tests {
		viewCalled = false
		middleWareCounter = 0

		t.Run(tt.name, func(t *testing.T) {
			s := New(&Config{LogLevel: "error"})
			s.UseMiddleware(tt.args.middlewareFns...)
			s.Path("GET", "/", tt.args.viewFn)

			ln := fasthttputil.NewInmemoryListener()

			serverCh := make(chan error, 1)
			go func() {
				err := s.serve(ln)
				serverCh <- err
			}()

			clientCh := make(chan struct{})
			go func() {
				c, err := ln.Dial()
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				if _, err = c.Write([]byte("GET / HTTP/1.1\r\nHost: TestServer\r\n\r\n")); err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				br := bufio.NewReader(c)
				var resp fasthttp.Response
				if err = resp.Read(br); err != nil {
					t.Fatalf("unexpected error: %s", err)
				}

				if resp.StatusCode() != tt.want.statusCode {
					t.Fatalf("Unexpected status code: '%d', want '%d'", resp.StatusCode(), tt.want.statusCode)
				}

				if viewCalled != tt.want.viewCalled {
					t.Errorf("View called = %v, want %v", viewCalled, tt.want.viewCalled)
				}

				if middleWareCounter != tt.want.middleWareCounter {
					t.Errorf("Middleware call counter = %v, want %v", middleWareCounter, tt.want.middleWareCounter)
				}

				if err = c.Close(); err != nil {
					t.Fatalf("unexpected error: %s", err)
				}

				close(clientCh)
			}()

			select {
			case <-clientCh:
			case <-time.After(time.Second):
				t.Fatalf("timeout")
			}

			if err := ln.Close(); err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			select {
			case <-serverCh:
			case <-time.After(time.Second):
				t.Fatalf("timeout")
			}
		})
	}
}

func TestAtreugo_getListener(t *testing.T) {
	type args struct {
		addr string
	}
	type want struct {
		addr    string
		network string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Test",
			args: args{
				addr: "127.0.0.1:8000",
			},
			want: want{
				addr:    "127.0.0.1:8000",
				network: "tcp",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(&Config{})

			ln := s.getListener(tt.args.addr)

			lnAddress := ln.Addr().String()
			if lnAddress != tt.want.addr {
				t.Errorf("Listener address: '%s', want '%s'", lnAddress, tt.want.addr)
			}

			lnNetwork := ln.Addr().Network()
			if lnNetwork != tt.want.network {
				t.Errorf("Listener network: '%s', want '%s'", lnNetwork, tt.want.network)
			}
		})
	}
}
