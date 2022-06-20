package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/extension"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/savsgio/atreugo/v11"
	"github.com/valyala/fasthttp"
)

var RSocketRegistry = make(map[string]*flux.Flux)

func RegisterProcessor(processor *flux.Flux) (string, error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	RSocketRegistry[uuid.String()] = processor
	return uuid.String(), nil
}

func RegisterProcessorWithID(id string, processor *flux.Flux) error {
	RSocketRegistry[id] = processor
	return nil
}

func UnRegisterProcessor(uuid string) {
	delete(RSocketRegistry, uuid)
}

func streamEvents(req payload.Payload) flux.Flux {

	log.Println("Recieved request")
	data := req.Data()
	uuid := strings.Trim(string(data), `"`)
	flx := RSocketRegistry[uuid]
	log.Println("Rsocket Registry", RSocketRegistry)
	if flx != nil {
		log.Println("Sending events for ", uuid)
		return *flx
	} else {
		log.Println("No events for ", uuid)
		return flux.Empty()
	}
}

func RsocketServer(port int) {
	log.Println("Starting RSocket server on port ", port)
	err := rsocket.Receive().
		Acceptor(func(ctx context.Context, setup payload.SetupPayload, sendingSocket rsocket.CloseableRSocket) (rsocket.RSocket, error) {
			// bind responder
			return rsocket.NewAbstractSocket(
				rsocket.RequestStream(streamEvents),
			), nil
		}).
		Transport(rsocket.WebsocketServer().SetAddr(fmt.Sprintf(":%d", port)).Build()).
		Serve(context.Background())
	if err != nil {
		log.Panicln("Failed to Websocket server ", err)
	}

}

func main() {
	host := flag.String("host", "", "Host")
	port := flag.Int("port", 8080, "Server Port")
	rsocketPort := flag.Int("rsocket-port", 9090, "RSocket Server Port")
	httpContextpath := flag.String("context-path", "", "Web server context path")

	address := fmt.Sprintf("%s:%d", *host, *port)

	config := atreugo.Config{
		Addr: address,
	}

	server := atreugo.New(config)
	contextPath := strings.TrimSuffix(*httpContextpath, "/")
	api := server.NewGroupPath(contextPath + "/api")
	api.GET("/configuration", func(rc *atreugo.RequestCtx) error {
		servConfig := map[string]interface{}{}
		servConfig["rsocketPort"] = *rsocketPort
		return rc.JSONResponse(servConfig, fasthttp.StatusOK)
	})
	api.GET("/trigger", func(rc *atreugo.RequestCtx) error {
		args := rc.QueryArgs()
		tabs := args.GetUintOrZero("tabs")
		nums := args.GetUintOrZero("nums")
		if tabs == 0 {
			tabs = 1
		}
		if nums == 0 {
			nums = 10
		}
		ids := make([]string, tabs)
		for i := 0; i < tabs; i++ {
			mychan := make(chan payload.Payload, 1000)
			err1 := make(chan error)
			fx := flux.CreateFromChannel(mychan, err1)

			id, err := RegisterProcessor(&fx)
			if err != nil {
				return err
			}
			ids[i] = id
			go func(id string, myChan1 chan<- payload.Payload, err2 chan<- error, size int) {

				j := 0
				for range time.Tick(1 * time.Second) {
					val := j + 1
					data := fmt.Sprintf(`{"id": "%s", "data": %v}`, id, val)
					log.Println("id:", id, "with value", val)
					myChan1 <- payload.New([]byte(data), []byte(extension.ApplicationJSON.String()))
					j++
					if j == size {
						break
					}
				}
				close(myChan1)
				close(err2)

			}(id, mychan, err1, nums)
		}
		return rc.JSONResponse(ids)
	})
	if contextPath != "" {
		server.ANY("/", func(rc *atreugo.RequestCtx) error {
			return rc.RedirectResponse(contextPath, fasthttp.StatusTemporaryRedirect)
		})
	}
	go RsocketServer(*rsocketPort)

	log.Println("Starting server on port ", *port)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
