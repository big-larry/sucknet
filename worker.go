package sucknet

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"sync"
	"time"
)

func NewWorker(ctx context.Context, limit int, listener net.Listener, handler func(net.Conn) error, errhandler func(c net.Conn, err error), gettimeout func(n int) time.Duration) error {
	pool1 := make(chan net.Conn)
	pool2 := make(chan net.Conn)
	pool_limiter := make(chan struct{}, limit)
	wg := sync.WaitGroup{}
	var err error

	f := func(c net.Conn) {
		if c == nil {
			return
		}
		if err = handler(c); err != nil {
			errhandler(c, err)
		}
		if err = c.Close(); err != nil {
			errhandler(c, err)
		}
	}

	addworker := func(pool chan net.Conn, timeout time.Duration) {
		//fmt.Println("add worker with timeout", timeout)
		go func() {
			wg.Add(1)
			var timer *time.Timer
			if timeout > 0 {
				timer = time.NewTimer(timeout)
			} else {
				timer = &time.Timer{}
			}
		loop:
			for {
				select {
				case <-ctx.Done():
					break loop
				case <-timer.C:
					break loop
				case c := <-pool:
					//fmt.Println("handling with", timeout)
					f(c)
					if timeout > 0 {
						timer.Reset(timeout)
					}
				}
			}
			//fmt.Println("done worker with timeout", timeout)
			wg.Done()
			<-pool_limiter
		}()
	}

	addworker(pool1, 0)
	addworker(pool2, 0)

	done := make(chan struct{}, 1)
	go func() {
		for {
			con, err := listener.Accept()
			if err != nil {
				errhandler(con, err)
				if errors.Is(err, net.ErrClosed) {
					break
				}
				continue
			}
			if err = con.(*tls.Conn).Handshake(); err != nil {
				errhandler(con, err)
				if err = con.Close(); err != nil {
					errhandler(con, err)
				}
				continue
			}

		loop:
			for {
				select {
				case pool1 <- con:
					break loop
				case pool2 <- con:
					break loop
				default:
					select {
					case pool_limiter <- struct{}{}:
						addworker(pool2, gettimeout(len(pool_limiter)))
						pool2 <- con
						break loop
					}
				}
			}
		}
		done <- struct{}{}
	}()
	<-ctx.Done()
	err = listener.Close()
	<-done
	close(pool1)
	close(pool2)
	wg.Wait()
	return err
}
