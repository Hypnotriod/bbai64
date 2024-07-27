package main

import (
	"bbai64/gstpipeline"
	"bbai64/streamer"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/Hypnotriod/jpegenc"
)

const SERVER_ADDRESS = ":1337"
const FRAMES_BUFFER_SIZE = 32
const MJPEG_FRAME_BOUNDARY = "frameboundary"
const CONNECTION_TIMEOUT = 1 * time.Second
const CAMERA_WIDTH = 1920
const CAMERA_HEIGHT = 1080
const RESCALE_WIDTH = 640  //1280
const RESCALE_HEIGHT = 360 //720

type PixelsRGB16 []byte

var jpegParams = jpegenc.EncodeParams{
	QualityFactor: jpegenc.QualityFactorMedium,
	PixelType:     jpegenc.PixelTypeRGB565,
	Subsample:     jpegenc.Subsample424,
}

func serveTcpRgb16StreamSocket(width int, height int, strmr *streamer.Streamer[PixelsRGB16], address string) {
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
		serveTcpRgb16StreamSocketConnection(conn, width, height, strmr, address)
		conn.Close()
	}
}

func serveTcpRgb16StreamSocketConnection(conn net.Conn, width int, height int, strmr *streamer.Streamer[PixelsRGB16], address string) {
	log.Print("Accepted input stream at ", address)

	buffer := [FRAMES_BUFFER_SIZE]PixelsRGB16{}
	for i := range buffer {
		buffer[i] = make(PixelsRGB16, width*height*2)
	}

	var buffIndex int32
	for {
		frame := buffer[buffIndex]
		buffIndex = (buffIndex + 1) % FRAMES_BUFFER_SIZE
		size, err := io.ReadFull(conn, frame)
		if err != nil {
			if err == io.EOF {
				log.Print("Socket connection closed at ", address)
			} else if size != len(frame) {
				log.Print("Stream has wrong rgba frame size! Expected: ", len(frame), ", but got ", size)
			} else {
				log.Print("Socket read error ", err)
			}
			break
		}
		if !strmr.Broadcast(&frame) {
			break
		}
	}
}

func handleMjpegStreamRequest(width int, height int, strmrL *streamer.Streamer[PixelsRGB16], strmrR *streamer.Streamer[PixelsRGB16]) func(w http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		log.Print("HTTP Connection established with ", req.RemoteAddr)
		rw.Header().Add("Content-Type", "multipart/x-mixed-replace; boundary=--"+MJPEG_FRAME_BOUNDARY)
		boundary := "\r\n--" + MJPEG_FRAME_BOUNDARY + "\r\nContent-Type: image/jpeg\r\n\r\n"

		clientL := strmrL.NewClient(FRAMES_BUFFER_SIZE/2 - 2)
		defer clientL.Close()
		clientR := strmrR.NewClient(FRAMES_BUFFER_SIZE/2 - 2)
		defer clientR.Close()
		timer := time.NewTimer(CONNECTION_TIMEOUT)
		defer timer.Stop()

		var frameL *PixelsRGB16
		var frameR *PixelsRGB16
		mergedFrame := make(PixelsRGB16, width*height*2*2)
		jpegBuffer := make([]byte, width*height)
		var ok bool
		for {
			select {
			case <-timer.C:
				log.Print("Lost stream for ", req.RemoteAddr)
				return
			case frameL, ok = <-clientL.C:
				for ok && len(clientL.C) != 0 {
					frameL, ok = <-clientL.C
				}
			}
			if !ok {
				break
			}
			select {
			case <-timer.C:
				log.Print("Lost stream for ", req.RemoteAddr)
				return
			case frameR, ok = <-clientR.C:
				for ok && len(clientR.C) != 0 {
					frameR, ok = <-clientR.C
				}
			}
			if !ok {
				break
			}
			timer.Reset(CONNECTION_TIMEOUT)

			copy(mergedFrame[:len(*frameL)], *frameL)
			copy(mergedFrame[len(*frameL):], *frameR)

			if n, err := io.WriteString(rw, boundary); err != nil || n != len(boundary) {
				log.Print("Cannot write response to ", req.RemoteAddr, ": ", err)
				break
			}
			bytesEncoded, err := jpegenc.Encode(width, height, jpegParams, mergedFrame, jpegBuffer)
			if err != nil {
				log.Print("Cannot write response to ", req.RemoteAddr, ": ", err)
				break
			}
			if n, err := rw.Write(jpegBuffer[:bytesEncoded]); n != bytesEncoded || err != nil {
				log.Print("Cannot write response to ", req.RemoteAddr, ": ", err)
				break
			}
			if n, err := io.WriteString(rw, "\r\n"); err != nil || n != 2 {
				log.Print("Cannot write response to ", req.RemoteAddr, ": ", err)
				break
			}
		}
		log.Print("HTTP Connection closed with ", req.RemoteAddr)
	}
}

func makeStereoCameraStreamer(inputAddrL string, inputAddrR string, outputAddr string) {
	strmrL := streamer.NewStreamer[PixelsRGB16](FRAMES_BUFFER_SIZE/2 - 2).Run()
	strmrR := streamer.NewStreamer[PixelsRGB16](FRAMES_BUFFER_SIZE/2 - 2).Run()
	go serveTcpRgb16StreamSocket(RESCALE_WIDTH, RESCALE_HEIGHT, strmrL, inputAddrL)
	go serveTcpRgb16StreamSocket(RESCALE_WIDTH, RESCALE_HEIGHT, strmrR, inputAddrR)
	http.HandleFunc(outputAddr, handleMjpegStreamRequest(RESCALE_WIDTH, RESCALE_HEIGHT*2, strmrL, strmrR))
}

func main() {
	// open with mjpeg_stream_stereo_comb.html
	makeStereoCameraStreamer(":9990", ":9991", "/mjpeg_stream")
	// gst-launch-1.0 videotestsrc ! video/x-raw, width=640, height=480, format=NV12 ! videoconvert ! video/x-raw, format=RGB16 ! tcpclientsink host=127.0.0.1 port=9990
	go gstpipeline.LauchImx219CsiCameraRgb16Stream(
		0, CAMERA_WIDTH, CAMERA_HEIGHT, RESCALE_WIDTH, RESCALE_HEIGHT, 9990)
	// gst-launch-1.0 videotestsrc ! video/x-raw, width=640, height=480, format=NV12 ! videoconvert ! video/x-raw, format=RGB16 ! tcpclientsink host=127.0.0.1 port=9991
	go gstpipeline.LauchImx219CsiCameraRgb16Stream(
		1, CAMERA_WIDTH, CAMERA_HEIGHT, RESCALE_WIDTH, RESCALE_HEIGHT, 9991)
	http.Handle("/", http.FileServer(http.Dir("./public")))
	http.ListenAndServe(SERVER_ADDRESS, nil)
}
