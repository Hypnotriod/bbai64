package main

import (
	"bbai64/muxer"
	"bufio"
	"image"
	"image/jpeg"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

const SERVER_ADDRESS = ":1337"
const BUFFERED_FRAMES_COUNT = 30
const MJPEG_FRAME_BOUNDARY = "frameboundary"
const CONNECTION_TIMEOUT = 1 * time.Second
const JPEG_QUALITY = 50

func serveTcpRgbaStreamSocket(fWidth int, fHeight int, mux *muxer.Muxer[image.RGBA], address string) {
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
		serveTcpRgbaStreamSocketConnection(conn, fWidth, fHeight, mux, address)
		conn.Close()
	}
}

func serveTcpRgbaStreamSocketConnection(conn net.Conn, fWidth int, fHeight int, mux *muxer.Muxer[image.RGBA], address string) {
	log.Print("Accepted input stream at ", address)

	buffer := [BUFFERED_FRAMES_COUNT]*image.RGBA{}
	for i := range buffer {
		buffer[i] = image.NewRGBA(image.Rect(0, 0, fWidth, fHeight))
	}

	var buffIndex int32
	reader := bufio.NewReader(conn)
	for {
		frame := buffer[buffIndex]
		buffIndex = (buffIndex + 1) % BUFFERED_FRAMES_COUNT
		size, err := reader.Read(frame.Pix)
		if err != nil {
			if err == io.EOF {
				log.Print("Socket connection closed at ", address)
			} else {
				log.Print("Socket read error ", err)
			}
			break
		}
		if size != len(frame.Pix) {
			log.Print("Stream has wrong rgba frame size! Expected: ", len(frame.Pix), ", but got ", size)
			break
		}
		mux.Broadcast <- frame
	}
}

func handleMjpegStreamRequest(mux *muxer.Muxer[image.RGBA]) func(w http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		log.Print("HTTP Connection established with ", req.RemoteAddr)
		rw.Header().Add("Content-Type", "multipart/x-mixed-replace; boundary=--"+MJPEG_FRAME_BOUNDARY)
		boundary := "\r\n--" + MJPEG_FRAME_BOUNDARY + "\r\nContent-Type: image/jpeg\r\n\r\n"
		jpegOpts := &jpeg.Options{Quality: JPEG_QUALITY}
		client := muxer.NewClient(mux)
		defer client.Close()
		timer := time.NewTimer(CONNECTION_TIMEOUT)
		defer timer.Stop()
		var frame *image.RGBA
		var ok bool
		for {
			select {
			case <-timer.C:
				log.Print("Lost stream for ", req.RemoteAddr)
				return
			case frame, ok = <-client.Send:
			}
			timer.Reset(CONNECTION_TIMEOUT)
			if !ok {
				break
			}
			if n, err := io.WriteString(rw, boundary); err != nil || n != len(boundary) {
				log.Print("Cannot write response to ", req.RemoteAddr)
				break
			}
			if err := jpeg.Encode(rw, frame, jpegOpts); err != nil {
				log.Print("Cannot write response to ", req.RemoteAddr)
				break
			}
			if n, err := io.WriteString(rw, "\r\n"); err != nil || n != 2 {
				log.Print("Cannot write response to ", req.RemoteAddr)
				break
			}
		}

		log.Print("HTTP Connection closed with ", req.RemoteAddr)
	}
}

func makeStereoCameraMuxer(inputAddr1 string, outputAddr string) {
	mux1 := muxer.NewMuxer[image.RGBA](BUFFERED_FRAMES_COUNT)
	go mux1.Run()
	// gst-launch-1.0 videotestsrc ! video/x-raw, width=640, height=480, format=NV12 ! videoconvert ! video/x-raw, format=RGBA ! tcpclientsink host=127.0.0.1 port=9990
	go serveTcpRgbaStreamSocket(640, 480, mux1, inputAddr1)
	http.HandleFunc(outputAddr, handleMjpegStreamRequest(mux1))
}

func main() {
	makeStereoCameraMuxer(":9990", "/mjpeg_stream")

	http.Handle("/", http.FileServer(http.Dir("./public")))

	http.ListenAndServe(SERVER_ADDRESS, nil)
}
