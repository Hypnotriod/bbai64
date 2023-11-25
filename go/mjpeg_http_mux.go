package main

import (
	"bbai64/gstpipeline"
	"bbai64/muxer"
	"bufio"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

const SERVER_ADDRESS = ":1337"
const CHUNKS_BUFFER_SIZE = 1024
const MJPEG_FRAME_BOUNDARY = "frameboundary"
const CONNECTION_TIMEOUT = 1 * time.Second
const CAMERA_WIDTH = 1920
const CAMERA_HEIGHT = 1080
const RESCALE_WIDTH = 1280
const RESCALE_HEIGHT = 720
const JPEG_QUALITY = 50

func serveTcpSocket(mux *muxer.Muxer, address string) {
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
		serveTcpSocketConnection(conn, mux, address)
		conn.Close()
	}
}

func serveTcpSocketConnection(conn net.Conn, mux *muxer.Muxer, address string) {
	log.Print("Accepted input stream at ", address)
	var buffIndex int32
	buffer := [CHUNKS_BUFFER_SIZE]muxer.Chunk{}
	reader := bufio.NewReader(conn)
	for {
		chunk := &buffer[buffIndex]
		buffIndex = (buffIndex + 1) % CHUNKS_BUFFER_SIZE
		size, err := reader.Read(chunk.Data[:])
		if err != nil {
			if err == io.EOF {
				log.Print("Socket connection closed at ", address)
			} else {
				log.Print("Socket read error ", err)
			}
			break
		}
		chunk.Size = size
		mux.Broadcast <- chunk
	}
}

func handleMjpegStreamRequest(mux *muxer.Muxer) func(w http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		log.Print("HTTP Connection established with ", req.RemoteAddr)
		rw.Header().Add("Content-Type", "multipart/x-mixed-replace; boundary=--"+MJPEG_FRAME_BOUNDARY)
		client := muxer.NewClient(mux)
		defer client.Close()
		timer := time.NewTimer(CONNECTION_TIMEOUT)
		defer timer.Stop()
		var chunk *muxer.Chunk
		var ok bool
		for {
			select {
			case <-timer.C:
				log.Print("Lost stream for ", req.RemoteAddr)
				return
			case chunk, ok = <-client.Send:
			}
			timer.Reset(CONNECTION_TIMEOUT)
			if !ok {
				break
			}
			_, err := rw.Write(chunk.Data[:chunk.Size])
			if err != nil {
				log.Print("Cannot write response to ", req.RemoteAddr)
				break
			}
		}

		log.Print("HTTP Connection closed with ", req.RemoteAddr)
	}
}

func makeMjpegMuxer(inputAddr string, outputAddr string) {
	mux := muxer.NewMuxer(CHUNKS_BUFFER_SIZE)
	go mux.Run()
	go serveTcpSocket(mux, inputAddr)
	http.HandleFunc(outputAddr, handleMjpegStreamRequest(mux))
}

func main() {
	makeMjpegMuxer(":9990", "/mjpeg_stream1")
	makeMjpegMuxer(":9991", "/mjpeg_stream2")

	go gstpipeline.LauchImx219CsiCameraMjpegStream(
		0, CAMERA_WIDTH, CAMERA_HEIGHT, RESCALE_WIDTH, RESCALE_HEIGHT, JPEG_QUALITY, MJPEG_FRAME_BOUNDARY, 9990)
	go gstpipeline.LauchImx219CsiCameraMjpegStream(
		1, CAMERA_WIDTH, CAMERA_HEIGHT, RESCALE_WIDTH, RESCALE_HEIGHT, JPEG_QUALITY, MJPEG_FRAME_BOUNDARY, 9991)

	http.Handle("/", http.FileServer(http.Dir("./public")))

	http.ListenAndServe(SERVER_ADDRESS, nil)
}
