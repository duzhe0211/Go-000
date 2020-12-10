package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

type slowHandler struct{}

const aLongTime = 5 * time.Second

func (_ slowHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	time.Sleep(aLongTime)
	fmt.Fprintf(w, "Hello world\n")
}

func groupHttpServe(server *http.Server, eg *errgroup.Group, ctx context.Context) {
	eg.Go(func() error {
		<-ctx.Done()
		shutdownCtx, _ := context.WithTimeout(context.Background(), aLongTime)
		fmt.Println("Done received", server.Addr)
		err := server.Shutdown(shutdownCtx)
		fmt.Println("Shutdown success", server.Addr)
		return err
	})
	eg.Go(func() error {
		fmt.Println("ListenAndServe", server.Addr)
		return server.ListenAndServe()
	})
}

var ErrGotSignal = errors.New("Got signal")

func groupWaitSignal(group *errgroup.Group, ctx context.Context) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	group.Go(func() error {
		select {
		case s := <-c:
			fmt.Println("Got signal: ", s)
			return errors.Wrapf(ErrGotSignal, "", s)
		case <-ctx.Done():
			return nil
		}
	})
}

func main() {
	eg, ctx := errgroup.WithContext(context.Background())

	server1 := &http.Server{
		//Addr: "127.0.0.1:80000", //INVALID ADDRESS; use this case, server1 will listen faild and whole progress will exit
		Addr:    "127.0.0.1:8080",
		Handler: slowHandler{},
	}
	server2 := &http.Server{
		Addr:    "127.0.0.1:8090",
		Handler: slowHandler{},
	}
	groupHttpServe(server1, eg, ctx)
	groupHttpServe(server2, eg, ctx)
	groupWaitSignal(eg, ctx)
	if err := eg.Wait(); err != nil {
		fmt.Println(err.Error())
	}
}
