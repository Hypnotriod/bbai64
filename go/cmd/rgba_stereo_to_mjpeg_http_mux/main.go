package main

import (
	"bbai64/gstpipeline"
	"bbai64/muxer"
	"image"
	"image/jpeg"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

const SERVER_ADDRESS = ":1337"
const BUFFERED_FRAMES_COUNT = 8
const MJPEG_FRAME_BOUNDARY = "frameboundary"
const CONNECTION_TIMEOUT = 1 * time.Second
const CAMERA_WIDTH = 1920
const CAMERA_HEIGHT = 1080
const RESCALE_WIDTH = 1280
const RESCALE_HEIGHT = 720
const JPEG_QUALITY = 50

func serveTcpRgbaStreamSocket(width int, height int, mux *muxer.Muxer[image.RGBA], address string) {
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
		serveTcpRgbaStreamSocketConnection(conn, width, height, mux, address)
		conn.Close()
	}
}

func serveTcpRgbaStreamSocketConnection(conn net.Conn, width int, height int, mux *muxer.Muxer[image.RGBA], address string) {
	log.Print("Accepted input stream at ", address)

	buffer := [BUFFERED_FRAMES_COUNT]*image.RGBA{}
	for i := range buffer {
		buffer[i] = image.NewRGBA(image.Rect(0, 0, width, height))
	}

	var buffIndex int32
	for {
		frame := buffer[buffIndex]
		buffIndex = (buffIndex + 1) % BUFFERED_FRAMES_COUNT
		size, err := io.ReadFull(conn, frame.Pix)
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

func handleMjpegStreamRequest(width int, height int, muxL *muxer.Muxer[image.RGBA], muxR *muxer.Muxer[image.RGBA]) func(w http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		log.Print("HTTP Connection established with ", req.RemoteAddr)
		rw.Header().Add("Content-Type", "multipart/x-mixed-replace; boundary=--"+MJPEG_FRAME_BOUNDARY)
		boundary := "\r\n--" + MJPEG_FRAME_BOUNDARY + "\r\nContent-Type: image/jpeg\r\n\r\n"
		jpegOpts := &jpeg.Options{Quality: JPEG_QUALITY}

		clientL := muxer.NewClient(muxL)
		defer clientL.Close()
		clientR := muxer.NewClient(muxR)
		defer clientR.Close()
		timer := time.NewTimer(CONNECTION_TIMEOUT)
		defer timer.Stop()

		var frameL *image.RGBA
		var frameR *image.RGBA
		imgCombined := image.NewRGBA(image.Rect(0, 0, width, height))
		var ok bool
		for {
			select {
			case <-timer.C:
				log.Print("Lost stream for ", req.RemoteAddr)
				return
			case frameL, ok = <-clientL.Send:
				for len(clientL.Send) != 0 {
					frameL, ok = <-clientL.Send
				}
			}
			if !ok {
				break
			}
			select {
			case <-timer.C:
				log.Print("Lost stream for ", req.RemoteAddr)
				return
			case frameR, ok = <-clientR.Send:
				for len(clientR.Send) != 0 {
					frameR, ok = <-clientR.Send
				}
			}
			if !ok {
				break
			}
			timer.Reset(CONNECTION_TIMEOUT)

			copy(imgCombined.Pix[:len(frameL.Pix)], frameL.Pix)
			copy(imgCombined.Pix[len(frameL.Pix):], frameR.Pix)

			if n, err := io.WriteString(rw, boundary); err != nil || n != len(boundary) {
				log.Print("Cannot write response to ", req.RemoteAddr)
				break
			}
			if err := jpeg.Encode(rw, imgCombined, jpegOpts); err != nil {
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

func makeStereoCameraMuxer(inputAddrL string, inputAddrR string, outputAddr string) {
	muxL := muxer.NewMuxer[image.RGBA](BUFFERED_FRAMES_COUNT)
	muxR := muxer.NewMuxer[image.RGBA](BUFFERED_FRAMES_COUNT)
	go muxL.Run()
	go muxR.Run()
	// gst-launch-1.0 videotestsrc ! video/x-raw, width=640, height=480, format=NV12 ! videoconvert ! video/x-raw, format=RGBA ! tcpclientsink host=127.0.0.1 port=9990
	go serveTcpRgbaStreamSocket(RESCALE_WIDTH, RESCALE_HEIGHT, muxL, inputAddrL)
	// gst-launch-1.0 videotestsrc ! video/x-raw, width=640, height=480, format=NV12 ! videoconvert ! video/x-raw, format=RGBA ! tcpclientsink host=127.0.0.1 port=9991
	go serveTcpRgbaStreamSocket(RESCALE_WIDTH, RESCALE_HEIGHT, muxR, inputAddrR)
	http.HandleFunc(outputAddr, handleMjpegStreamRequest(RESCALE_WIDTH, RESCALE_HEIGHT*2, muxL, muxR))
}

func main() {
	makeStereoCameraMuxer(":9990", ":9991", "/mjpeg_stream")
	go gstpipeline.LauchImx219CsiCameraRgbaStream(
		0, CAMERA_WIDTH, CAMERA_HEIGHT, RESCALE_WIDTH, RESCALE_HEIGHT, 9990)
	go gstpipeline.LauchImx219CsiCameraRgbaStream(
		1, CAMERA_WIDTH, CAMERA_HEIGHT, RESCALE_WIDTH, RESCALE_HEIGHT, 9991)
	http.Handle("/", http.FileServer(http.Dir("./public")))
	http.ListenAndServe(SERVER_ADDRESS, nil)
}
