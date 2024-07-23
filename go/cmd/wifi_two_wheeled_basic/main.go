package main

import (
	"bbai64/gstpipeline"
	"bbai64/i2c"
	"bbai64/streamer"
	"bbai64/twowheeled"
	"bbai64/ups"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
)

var json jsoniter.API = jsoniter.ConfigCompatibleWithStandardLibrary

const SERVER_ADDRESS = ":1337"
const MJPEG_STREAM_CHUNKS_BUFFER_LENGTH = 1024
const MJPEG_STREAM_CHUNK_SIZE = 4096
const MJPEG_FRAME_BOUNDARY = "frameboundary"
const CONNECTION_TIMEOUT = 1 * time.Second
const CAMERA_WIDTH = 1920
const CAMERA_HEIGHT = 1080
const RESCALE_WIDTH = 1280
const RESCALE_HEIGHT = 720
const JPEG_QUALITY = 50

type Chunk struct {
	Data [MJPEG_STREAM_CHUNK_SIZE]byte
	Size int
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
	CheckOrigin:     checkOrigin,
}

type SystemStatus struct {
	Battery ups.UpsModuleStatus `json:"battery"`
}

var upsModule *ups.UpsModule3S
var wsMutex sync.Mutex

func checkOrigin(r *http.Request) bool {
	return true
}

func serveVehicleControlWSRequest(w http.ResponseWriter, r *http.Request) {
	if !wsMutex.TryLock() {
		log.Print("Websocket multiple connections are not allowed with ", r.Host)
		return
	}
	defer wsMutex.Unlock()
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Websocket upgrade error: ", err)
		return
	}
	log.Print("Websocket connection established with ", r.Host)
	defer conn.Close()
	vehicleState := &twowheeled.State{}
	systemStatus := &SystemStatus{}
	for {
		conn.SetReadDeadline(time.Now().Add(CONNECTION_TIMEOUT))
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Print("Websocket read error: ", err)
			break
		}
		err = json.Unmarshal(message, vehicleState)
		if err != nil {
			log.Print("Websocket command format error: ", err)
			break
		}
		twowheeled.UpdateWithState(vehicleState)
		systemStatus.Battery = upsModule.Status()
		message, _ = json.Marshal(systemStatus)
		err = conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Print("Websocket write error: ", err)
			break
		}
	}
	twowheeled.Reset()
	log.Print("Websocket connection terminated with ", r.Host)
}

func serveMjpegStreamTcpSocket(strmr *streamer.Streamer[Chunk], address string) {
	soc, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal("Cannot open socket at ", address, " : ", err)
	}
	for {
		log.Print("Waiting for input stream at ", address)
		conn, err := soc.Accept()
		if err != nil {
			log.Fatal("Cannot accept socket connection at ", address, " : ", err)
		}
		serveMjpegStreamTcpSocketConnection(conn, strmr, address)
		conn.Close()
	}
}

func serveMjpegStreamTcpSocketConnection(conn net.Conn, strmr *streamer.Streamer[Chunk], address string) {
	log.Print("Accepted input stream at ", address)
	var buffIndex int32
	buffer := [MJPEG_STREAM_CHUNKS_BUFFER_LENGTH]Chunk{}
	for {
		chunk := &buffer[buffIndex]
		buffIndex = (buffIndex + 1) % MJPEG_STREAM_CHUNKS_BUFFER_LENGTH
		size, err := conn.Read(chunk.Data[:])
		if err != nil {
			if err == io.EOF {
				log.Print("Socket connection closed at ", address)
			} else {
				log.Print("Socket read error: ", err)
			}
			break
		}
		chunk.Size = size
		if !strmr.Broadcast(chunk) {
			break
		}
	}
}

func handleMjpegStreamHttpRequest(strmr *streamer.Streamer[Chunk]) func(w http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		log.Print("HTTP Connection established with ", req.RemoteAddr)
		rw.Header().Add("Content-Type", "multipart/x-mixed-replace; boundary=--"+MJPEG_FRAME_BOUNDARY)

		client := strmr.NewClient(MJPEG_STREAM_CHUNKS_BUFFER_LENGTH/2 - 2)
		defer client.Close()
		timer := time.NewTimer(CONNECTION_TIMEOUT)
		defer timer.Stop()

		var chunk *Chunk
		var ok bool
		for {
			select {
			case <-timer.C:
				log.Print("Lost stream for ", req.RemoteAddr)
				return
			case chunk, ok = <-client.C:
			}
			timer.Reset(CONNECTION_TIMEOUT)
			if !ok {
				break
			}
			_, err := rw.Write(chunk.Data[:chunk.Size])
			if err != nil {
				log.Print("Cannot write response to ", req.RemoteAddr, " : ", err)
				break
			}
		}

		log.Print("HTTP Connection closed with ", req.RemoteAddr)
	}
}

func makeMjpegStreamer(inputAddr string, outputAddr string) *streamer.Streamer[Chunk] {
	strmr := streamer.NewStreamer[Chunk](MJPEG_STREAM_CHUNKS_BUFFER_LENGTH/2 - 2).Run()
	go serveMjpegStreamTcpSocket(strmr, inputAddr)
	http.HandleFunc(outputAddr, handleMjpegStreamHttpRequest(strmr))
	return strmr
}

func main() {
	upsModule = ups.NewUpsModule3S(i2c.Bus1)
	go upsModule.Run(time.Second)
	defer upsModule.Stop()

	twowheeled.Initialize()
	strmr := makeMjpegStreamer(":9990", "/mjpeg_stream")
	defer strmr.Stop()
	go gstpipeline.LauchImx219CsiCameraMjpegStream(
		0, CAMERA_WIDTH, CAMERA_HEIGHT, RESCALE_WIDTH, RESCALE_HEIGHT, JPEG_QUALITY, MJPEG_FRAME_BOUNDARY, 9990)

	http.HandleFunc("/ws", serveVehicleControlWSRequest)
	http.Handle("/", http.FileServer(http.Dir("./public")))
	if err := http.ListenAndServe(SERVER_ADDRESS, nil); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("Unable to start HTTP server: ", err)
	}
}
