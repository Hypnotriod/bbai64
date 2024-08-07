package main

import (
	"bbai64/gstpipeline"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
	"unsafe"

	"github.com/Hypnotriod/jpegenc"
	"github.com/Hypnotriod/streamer"
	tf "github.com/galeone/tensorflow/tensorflow/go"
	tg "github.com/galeone/tfgo"
)

const SERVER_ADDRESS = ":1337"
const FRAMES_BUFFER_SIZE = 64
const CHUNKS_BUFFER_SIZE = 1024
const CHUNK_SIZE = 4096
const MJPEG_STREAM_CHUNKS_BUFFER_LENGTH = 1024
const MJPEG_STREAM_CHUNK_SIZE = 4096
const MJPEG_FRAME_BOUNDARY = "frameboundary"
const JPEG_QUALITY = 50
const CONNECTION_TIMEOUT = 1 * time.Second
const CAMERA_WIDTH = 1920
const CAMERA_HEIGHT = 1080
const RESCALE_VISUALIZATION_WIDTH = 1280
const RESCALE_VISUALIZATION_HEIGHT = 720
const RESCALE_ANALYTICS_WIDTH = 320
const RESCALE_ANALYTICS_HEIGHT = 180
const TENSOR_WIDTH = 128
const TENSOR_HEIGHT = 128
const CHANNELS_NUM = 3
const TOP_PREDICTIONS_NUM = 5
const PREDICT_EACH_FRAME = 5

type PixelsRGB []byte

type Chunk struct {
	Data [CHUNK_SIZE]byte
	Size int
}

var inputTensor *tf.Tensor
var tensorInputFlat *[1 * TENSOR_WIDTH * TENSOR_HEIGHT * CHANNELS_NUM]float32
var model *tg.Model
var labels []string

var jpegParams = jpegenc.EncodeParams{
	QualityFactor: jpegenc.QualityFactorBest,
	PixelType:     jpegenc.PixelTypeRGB888,
	Subsample:     jpegenc.Subsample424,
	ChromaSwap:    true,
}

func serveAnalyticsStreamTcpSocket(width int, height int, strmr *streamer.Streamer[PixelsRGB], address string) {
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
		serveAnalyticsStreamTcpSocketConnection(conn, width, height, strmr, address)
		conn.Close()
	}
}

func serveAnalyticsStreamTcpSocketConnection(conn net.Conn, width int, height int, strmr *streamer.Streamer[PixelsRGB], address string) {
	log.Print("Accepted input stream at ", address)

	buffer := [FRAMES_BUFFER_SIZE]PixelsRGB{}
	for i := range buffer {
		buffer[i] = make(PixelsRGB, width*height*CHANNELS_NUM)
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
				log.Print("Stream has wrong frame size! Expected: ", len(frame), ", but got ", size)
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

func serveVisualizationMjpegStreamTcpSocket(strmr *streamer.Streamer[Chunk], address string) {
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
		serveVisualizationMjpegStreamTcpSocketConnection(conn, strmr, address)
		conn.Close()
	}
}

func serveVisualizationMjpegStreamTcpSocketConnection(conn net.Conn, strmr *streamer.Streamer[Chunk], address string) {
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

func handleVisualizationMjpegStreamRequest(strmr *streamer.Streamer[Chunk]) func(w http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		log.Print("HTTP Connection established with ", req.RemoteAddr)
		rw.Header().Add("Content-Type", "multipart/x-mixed-replace; boundary=--"+MJPEG_FRAME_BOUNDARY)

		consumer := strmr.NewConsumer(CHUNKS_BUFFER_SIZE/2 - 2)
		defer consumer.Close()
		timer := time.NewTimer(CONNECTION_TIMEOUT)
		defer timer.Stop()

		var chunk *Chunk
		var ok bool
		for {
			select {
			case <-timer.C:
				log.Print("Lost stream for ", req.RemoteAddr)
				return
			case chunk, ok = <-consumer.C:
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

func handleAnalyticsMjpegStreamRequest(width int, height int, strmr *streamer.Streamer[PixelsRGB]) func(w http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		log.Print("HTTP Connection established with ", req.RemoteAddr)
		rw.Header().Add("Content-Type", "multipart/x-mixed-replace; boundary=--"+MJPEG_FRAME_BOUNDARY)
		boundary := "\r\n--" + MJPEG_FRAME_BOUNDARY + "\r\nContent-Type: image/jpeg\r\n\r\n"

		consumer := strmr.NewConsumer(FRAMES_BUFFER_SIZE/2 - 2)
		defer consumer.Close()
		timer := time.NewTimer(CONNECTION_TIMEOUT)
		defer timer.Stop()

		var frame *PixelsRGB
		jpegBuffer := make([]byte, width*height)
		var ok bool
		for {
			select {
			case <-timer.C:
				log.Print("Lost stream for ", req.RemoteAddr)
				return
			case frame, ok = <-consumer.C:
				for ok && len(consumer.C) != 0 {
					frame, ok = <-consumer.C
				}
			}
			if !ok {
				break
			}
			timer.Reset(CONNECTION_TIMEOUT)

			if n, err := io.WriteString(rw, boundary); err != nil || n != len(boundary) {
				log.Print("Cannot write response to ", req.RemoteAddr, ": ", err)
				break
			}
			bytesEncoded, err := jpegenc.Encode(width, height, jpegParams, *frame, jpegBuffer)
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

func makeVisualizationMjpegStreamer(inputAddr string, outputAddr string) *streamer.Streamer[Chunk] {
	strmr := streamer.NewStreamer[Chunk](CHUNKS_BUFFER_SIZE/2 - 2).Run()
	go serveVisualizationMjpegStreamTcpSocket(strmr, inputAddr)
	http.HandleFunc(outputAddr, handleVisualizationMjpegStreamRequest(strmr))
	return strmr
}

func makeAnalyticsCameraStreamer(inputAddr string, outputAddr string) *streamer.Streamer[PixelsRGB] {
	strmr := streamer.NewStreamer[PixelsRGB](FRAMES_BUFFER_SIZE/2 - 2).Run()
	go serveAnalyticsStreamTcpSocket(TENSOR_WIDTH, TENSOR_HEIGHT, strmr, inputAddr)
	http.HandleFunc(outputAddr, handleAnalyticsMjpegStreamRequest(TENSOR_WIDTH, TENSOR_HEIGHT, strmr))
	return strmr
}

func initModel() {
	var err error
	inputTensor, err = tf.NewTensor([1][TENSOR_WIDTH][TENSOR_HEIGHT][CHANNELS_NUM]float32{})
	if err != nil {
		log.Fatal("Cannot create input tensor : ", err)
	}
	tensorInputFlat = (*[1 * TENSOR_WIDTH * TENSOR_HEIGHT * CHANNELS_NUM]float32)(unsafe.Pointer(&inputTensor.TensorData()[0]))

	model = tg.LoadModel("model/mobilenet_v2", []string{"serve"}, nil)
	labelsRaw, err := os.ReadFile("model/mobilenet_v2/labels.txt")
	if err != nil {
		log.Fatal("Cannot read model labels: ", err)
	}
	labels = strings.Split(string(labelsRaw), "\n")

	predict(false) // make first prediction beforehand to trigger model lazy loading
}

func predict(printPredictions bool) {
	startTime := time.Now()
	results := model.Exec([]tf.Output{
		model.Op("StatefulPartitionedCall", 0),
	}, map[tf.Output]*tf.Tensor{
		model.Op("serving_default_inputs", 0): inputTensor,
	})
	if printPredictions {
		predictions := results[0].Value().([][]float32)[0]
		printTopPredictions(TOP_PREDICTIONS_NUM, predictions, time.Since(startTime))
	}
}

func printTopPredictions(num int, predictions []float32, timeTaken time.Duration) {
	var topPredictions []float32 = make([]float32, num)
	var topLabels []string = make([]string, num)
	for i, label := range labels {
	jloop:
		for j := 0; j < len(topPredictions); j++ {
			if predictions[i] < topPredictions[j] {
				continue
			}
			for k := len(topPredictions) - 2; k >= j; k-- {
				topPredictions[k+1] = topPredictions[k]
				topLabels[k+1] = topLabels[k]
			}
			topPredictions[j] = predictions[i]
			topLabels[j] = label
			break jloop
		}
	}
	for i, label := range topLabels {
		fmt.Printf("%s: %.2f, ", label, topPredictions[i])
	}
	fmt.Println(timeTaken)
}

func feedFrame(frame []byte) {
	for i := 0; i < len(tensorInputFlat); i++ {
		tensorInputFlat[i] = float32(frame[i]) / 255.0
	}
}

func processFrames(strmr *streamer.Streamer[PixelsRGB]) {
	consumer := strmr.NewConsumer(FRAMES_BUFFER_SIZE/2 - 2)
	defer consumer.Close()
	for {
		for i := 0; i < PREDICT_EACH_FRAME-1; i++ { // skip frames
			if _, ok := <-consumer.C; !ok {
				return
			}
		}
		frame, ok := <-consumer.C
		if !ok {
			return
		}
		feedFrame(*frame)
		predict(true)
	}
}

func main() {
	initModel()

	strmrAnalytics := makeAnalyticsCameraStreamer(":9990", "/mjpeg_stream1")
	defer strmrAnalytics.Stop()

	strmrVisualization := makeVisualizationMjpegStreamer(":9991", "/mjpeg_stream2")
	defer strmrVisualization.Stop()

	go gstpipeline.LauchImx219CsiCameraAnalyticsRgbStream1VisualizationMjpegStream2(
		0, CAMERA_WIDTH, CAMERA_HEIGHT,
		RESCALE_ANALYTICS_WIDTH, RESCALE_ANALYTICS_HEIGHT,
		TENSOR_WIDTH, TENSOR_HEIGHT,
		9990,
		RESCALE_VISUALIZATION_WIDTH, RESCALE_VISUALIZATION_HEIGHT,
		JPEG_QUALITY,
		MJPEG_FRAME_BOUNDARY,
		9991)
	go processFrames(strmrAnalytics)

	http.Handle("/", http.FileServer(http.Dir("./public")))

	http.ListenAndServe(SERVER_ADDRESS, nil)
}
