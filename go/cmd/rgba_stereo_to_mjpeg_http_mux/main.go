package main

import (
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

func handleMjpegStreamRequest(rWidth int, rHeight int, mux1 *muxer.Muxer[image.RGBA], mux2 *muxer.Muxer[image.RGBA]) func(w http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		log.Print("HTTP Connection established with ", req.RemoteAddr)
		rw.Header().Add("Content-Type", "multipart/x-mixed-replace; boundary=--"+MJPEG_FRAME_BOUNDARY)
		boundary := "\r\n--" + MJPEG_FRAME_BOUNDARY + "\r\nContent-Type: image/jpeg\r\n\r\n"
		jpegOpts := &jpeg.Options{Quality: JPEG_QUALITY}

		client1 := muxer.NewClient(mux1)
		defer client1.Close()
		client2 := muxer.NewClient(mux2)
		defer client1.Close()
		timer := time.NewTimer(CONNECTION_TIMEOUT)
		defer timer.Stop()

		var frame1 *image.RGBA
		var frame2 *image.RGBA
		imgCombined := image.NewRGBA(image.Rect(0, 0, rWidth, rHeight))
		var ok bool
		for {
			select {
			case <-timer.C:
				log.Print("Lost stream for ", req.RemoteAddr)
				return
			case frame1, ok = <-client1.Send:
				for len(client1.Send) != 0 {
					frame1, ok = <-client1.Send
				}
			}
			if !ok {
				break
			}
			select {
			case <-timer.C:
				log.Print("Lost stream for ", req.RemoteAddr)
				return
			case frame2, ok = <-client2.Send:
				for len(client2.Send) != 0 {
					frame2, ok = <-client2.Send
				}
			}
			if !ok {
				break
			}
			timer.Reset(CONNECTION_TIMEOUT)

			copy(imgCombined.Pix[:len(frame1.Pix)], frame1.Pix)
			copy(imgCombined.Pix[len(frame1.Pix):], frame2.Pix)

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

func makeStereoCameraMuxer(inputAddr1 string, inputAddr2 string, outputAddr string) {
	mux1 := muxer.NewMuxer[image.RGBA](BUFFERED_FRAMES_COUNT)
	mux2 := muxer.NewMuxer[image.RGBA](BUFFERED_FRAMES_COUNT)
	go mux1.Run()
	go mux2.Run()
	// gst-launch-1.0 videotestsrc ! video/x-raw, width=640, height=480, format=NV12 ! videoconvert ! video/x-raw, format=RGBA ! tcpclientsink host=127.0.0.1 port=9990
	go serveTcpRgbaStreamSocket(640, 480, mux1, inputAddr1)
	// gst-launch-1.0 videotestsrc ! video/x-raw, width=640, height=480, format=NV12 ! videoconvert ! video/x-raw, format=RGBA ! tcpclientsink host=127.0.0.1 port=9991
	go serveTcpRgbaStreamSocket(640, 480, mux2, inputAddr2)
	http.HandleFunc(outputAddr, handleMjpegStreamRequest(640, 480*2, mux1, mux2))
}

func main() {
	makeStereoCameraMuxer(":9990", ":9991", "/mjpeg_stream")

	http.Handle("/", http.FileServer(http.Dir("./public")))

	http.ListenAndServe(SERVER_ADDRESS, nil)
}
