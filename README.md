# sucknet

```go
	listener, err := tls.Listen("tcp", ":8443", cfg)
	if err != nil {
		log.Fatalln(err)
	}

	sucknet.NewWorker(ctx, 10, listener, handler, func(c net.Conn, err error) {
		if c != nil {
			log.Println(c.RemoteAddr().String(), err)
		} else {
			log.Println(err)
		}
	}, func(n int) time.Duration {
		return time.Second * time.Duration(20-n)
	})
```
