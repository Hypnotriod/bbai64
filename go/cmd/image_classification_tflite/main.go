package main

import (
	"bbai64/gstpipeline"
	"bbai64/streamer"
	"bbai64/titfldelegate"
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Hypnotriod/jpegenc"
	"github.com/gorilla/websocket"
	"github.com/mattn/go-tflite"
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
const USE_IMX219_CSI_CAMERA = true
const CAMERA_INDEX = 0
const CAMERA_WIDTH = 1920
const CAMERA_HEIGHT = 1080
const RESCALE_VISUALIZATION_WIDTH = 1280
const RESCALE_VISUALIZATION_HEIGHT = 720
const RESCALE_ANALYTICS_WIDTH = 426
const RESCALE_ANALYTICS_HEIGHT = 240
const TENSOR_WIDTH = 224
const TENSOR_HEIGHT = 224
const MEAN = 0
const SCALE = 1.0 / 255.0
const CHANNELS_NUM = 3
const TENSOR_SIZE = TENSOR_WIDTH * TENSOR_HEIGHT * CHANNELS_NUM
const TOP_PREDICTIONS_NUM = 5
const PREDICT_EACH_FRAME = 1
const USE_DELEGATE = true
const MODEL_PATH = "model/mobileNetV1-mlperf/model/mobilenet_v1_1.0_224.tflite"
const LABELS_PATH = "model/mobileNetV1-mlperf/labels.txt"
const ARTIFACTS_PATH = "model/mobileNetV1-mlperf/artifacts"
const TFL_DELEGATE_PATH = "/usr/lib/libtidl_tfl_delegate.so"

type PixelsRGB []byte

type Chunk struct {
	Data [CHUNK_SIZE]byte
	Size int
}

type Prediction struct {
	Label string  `json:"label"`
	Class int     `json:"class"`
	Score float32 `json:"score"`
}

type Predictions []Prediction

var interpreter *tflite.Interpreter
var labels []string

var jpegParams = jpegenc.EncodeParams{
	QualityFactor: jpegenc.QualityFactorBest,
	PixelType:     jpegenc.PixelTypeRGB888,
	Subsample:     jpegenc.Subsample424,
	ChromaSwap:    true,
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
	CheckOrigin:     checkOrigin,
}

func checkOrigin(r *http.Request) bool {
	return true
}

func serveInferenceResultWSRequest(strmr *streamer.Streamer[Predictions]) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("Websocket upgrade error: ", err)
			return
		}
		log.Print("Websocket connection established with ", r.Host)
		defer conn.Close()
		client := strmr.NewClient(streamer.BufferSizeFromTotal(FRAMES_BUFFER_SIZE))
		defer client.Close()
		for {
			predictions, ok := <-client.C
			if !ok {
				break
			}
			message, _ := json.Marshal(predictions)
			conn.SetWriteDeadline(time.Now().Add(CONNECTION_TIMEOUT))
			err = conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Print("Websocket write error: ", err)
				break
			}
		}
		log.Print("Websocket connection terminated with ", r.Host)
	}
}

func serveAnalyticsStreamTcpSocket(width int, height int, strmr *streamer.Streamer[PixelsRGB], address string) {
	soc, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal("Cannot open socket at ", address, " : ", err)
	}
	defer soc.Close()
	for {
		log.Print("Waiting for input stream at ", address)
		conn, err := soc.Accept()
		if err != nil {
			log.Fatal("Cannot accept socket connection at ", address, " : ", err)
		}
		serveAnalyticsStreamTcpSocketConnection(conn, width, height, strmr, address)
		conn.Close()
		if !strmr.IsRunning() {
			break
		}
	}
}

func serveAnalyticsStreamTcpSocketConnection(conn net.Conn, width int, height int, strmr *streamer.Streamer[PixelsRGB], address string) {
	log.Print("Accepted input stream at ", address)

	buffer := [FRAMES_BUFFER_SIZE]PixelsRGB{}
	for i := range buffer {
		buffer[i] = make(PixelsRGB, width*height*CHANNELS_NUM)
	}

	var buffIndex int
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
	defer soc.Close()
	for {
		log.Print("Waiting for input stream at ", address)
		conn, err := soc.Accept()
		if err != nil {
			log.Fatal("Cannot accept socket connection at ", address, " : ", err)
		}
		serveVisualizationMjpegStreamTcpSocketConnection(conn, strmr, address)
		conn.Close()
		if !strmr.IsRunning() {
			break
		}
	}
}

func serveVisualizationMjpegStreamTcpSocketConnection(conn net.Conn, strmr *streamer.Streamer[Chunk], address string) {
	log.Print("Accepted input stream at ", address)
	var buffIndex int
	buffer := [MJPEG_STREAM_CHUNKS_BUFFER_LENGTH]Chunk{}
	reader := bufio.NewReader(conn)
	for {
		chunk := &buffer[buffIndex]
		buffIndex = (buffIndex + 1) % MJPEG_STREAM_CHUNKS_BUFFER_LENGTH
		size, err := reader.Read(chunk.Data[:])
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

		client := strmr.NewClient(CHUNKS_BUFFER_SIZE/2 - 2)
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

		client := strmr.NewClient(streamer.BufferSizeFromTotal(FRAMES_BUFFER_SIZE))
		defer client.Close()
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
			case frame, ok = <-client.C:
				for ok && len(client.C) != 0 {
					frame, ok = <-client.C
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
	strmr := streamer.NewStreamer[Chunk](CHUNKS_BUFFER_SIZE/2 - 2)
	go strmr.Run()
	go serveVisualizationMjpegStreamTcpSocket(strmr, inputAddr)
	http.HandleFunc(outputAddr, handleVisualizationMjpegStreamRequest(strmr))
	return strmr
}

func makeAnalyticsCameraStreamer(inputAddr string, outputAddr string) *streamer.Streamer[PixelsRGB] {
	strmr := streamer.NewStreamer[PixelsRGB](streamer.BufferSizeFromTotal(FRAMES_BUFFER_SIZE))
	go strmr.Run()
	go serveAnalyticsStreamTcpSocket(TENSOR_WIDTH, TENSOR_HEIGHT, strmr, inputAddr)
	http.HandleFunc(outputAddr, handleAnalyticsMjpegStreamRequest(TENSOR_WIDTH, TENSOR_HEIGHT, strmr))
	return strmr
}

func initModel() *tflite.Model {
	model := tflite.NewModelFromFile(MODEL_PATH)
	if model == nil {
		log.Fatal("Cannot load model")
	}

	labelsRaw, err := os.ReadFile(LABELS_PATH)
	if err != nil {
		log.Fatal("Cannot read model labels: ", err)
	}
	labels = strings.Split(string(labelsRaw), "\n")
	for i := range labels {
		labels[i] = strings.Trim(labels[i], "\r")
	}

	options := tflite.NewInterpreterOptions()
	if USE_DELEGATE {
		delegate := titfldelegate.TiTflDelegateCreate(TFL_DELEGATE_PATH, ARTIFACTS_PATH)
		options.AddDelegate(delegate)
	}

	interpreter = tflite.NewInterpreter(model, options)
	if interpreter == nil {
		log.Fatal("Cannot create interpreter")
	}

	status := interpreter.AllocateTensors()
	if status != tflite.OK {
		log.Fatal("Tensor allocation failed")
	}
	return model
}

func processFramesUint8(frameStrmr *streamer.Streamer[PixelsRGB], predStrmr *streamer.Streamer[Predictions]) {
	inputTensor := (*[TENSOR_SIZE]byte)(interpreter.GetInputTensor(0).Data())
	buffer := [FRAMES_BUFFER_SIZE]Predictions{}
	client := frameStrmr.NewClient(streamer.BufferSizeFromTotal(FRAMES_BUFFER_SIZE))
	var buffIndex int
	defer client.Close()
	for {
		for i := 0; i < PREDICT_EACH_FRAME-1; i++ {
			if _, ok := <-client.C; !ok {
				return
			}
		}
		frame, ok := <-client.C
		if !ok {
			return
		}
		copy(inputTensor[:], *frame)
		predict(&buffer[buffIndex])
		if !predStrmr.Broadcast(&buffer[buffIndex]) {
			break
		}
		buffIndex = (buffIndex + 1) % FRAMES_BUFFER_SIZE
	}
}

func processFramesFloat32(frameStrmr *streamer.Streamer[PixelsRGB], predStrmr *streamer.Streamer[Predictions]) {
	inputTensor := (*[TENSOR_SIZE]float32)(interpreter.GetInputTensor(0).Data())
	buffer := [FRAMES_BUFFER_SIZE]Predictions{}
	client := frameStrmr.NewClient(streamer.BufferSizeFromTotal(FRAMES_BUFFER_SIZE))
	var buffIndex int
	defer client.Close()
	for {
		for i := 0; i < PREDICT_EACH_FRAME-1; i++ {
			if _, ok := <-client.C; !ok {
				return
			}
		}
		frame, ok := <-client.C
		if !ok {
			return
		}
		for i, b := range *frame {
			inputTensor[i] = (float32(b) - MEAN) * SCALE
		}
		predict(&buffer[buffIndex])
		if !predStrmr.Broadcast(&buffer[buffIndex]) {
			break
		}
		buffIndex = (buffIndex + 1) % FRAMES_BUFFER_SIZE
	}
}

func predict(predictions *Predictions) {
	startTime := time.Now()
	status := interpreter.Invoke()
	if status != tflite.OK {
		log.Println("Interpreter invoke failed")
		return
	}
	endTime := time.Since(startTime)
	result := interpreter.GetOutputTensor(0).Float32s()
	var topPredictions []float32 = make([]float32, TOP_PREDICTIONS_NUM)
	var topLabels []string = make([]string, TOP_PREDICTIONS_NUM)
	var topClasses []int = make([]int, TOP_PREDICTIONS_NUM)
	for i, label := range labels {
	jloop:
		for j := 0; j < len(topPredictions); j++ {
			if result[i] < topPredictions[j] {
				continue
			}
			for k := len(topPredictions) - 2; k >= j; k-- {
				topPredictions[k+1] = topPredictions[k]
				topLabels[k+1] = topLabels[k]
				topClasses[k+1] = topClasses[k]
			}
			topPredictions[j] = result[i]
			topLabels[j] = label
			topClasses[j] = i
			break jloop
		}
	}
	if cap(*predictions) == 0 {
		*predictions = make(Predictions, 0, TOP_PREDICTIONS_NUM)
	}
	*predictions = (*predictions)[:0]
	for i, label := range topLabels {
		fmt.Printf("%s: %.2f%%, ", label, topPredictions[i]*100)
		*predictions = append(*predictions, Prediction{
			Label: label,
			Class: topClasses[i],
			Score: topPredictions[i],
		})
	}
	fmt.Println(endTime)
}

func main() {
	server := &http.Server{Addr: SERVER_ADDRESS}
	model := initModel()
	analyticsStrmr := makeAnalyticsCameraStreamer(":9990", "/mjpeg_stream1")
	visualizationStrmr := makeVisualizationMjpegStreamer(":9991", "/mjpeg_stream2")

	if USE_IMX219_CSI_CAMERA {
		go gstpipeline.LauchImx219CsiCameraAnalyticsRgbStream1VisualizationMjpegStream2(
			CAMERA_INDEX, CAMERA_WIDTH, CAMERA_HEIGHT,
			RESCALE_ANALYTICS_WIDTH, RESCALE_ANALYTICS_HEIGHT,
			TENSOR_WIDTH, TENSOR_HEIGHT, 9990,
			RESCALE_VISUALIZATION_WIDTH, RESCALE_VISUALIZATION_HEIGHT,
			JPEG_QUALITY, MJPEG_FRAME_BOUNDARY, 9991)
	} else {
		go gstpipeline.LauchUsbJpegCameraAnalyticsRgbStream1VisualizationMjpegStream2(
			CAMERA_INDEX, CAMERA_WIDTH, CAMERA_HEIGHT,
			RESCALE_ANALYTICS_WIDTH, RESCALE_ANALYTICS_HEIGHT,
			TENSOR_WIDTH, TENSOR_HEIGHT, 9990,
			RESCALE_VISUALIZATION_WIDTH, RESCALE_VISUALIZATION_HEIGHT,
			JPEG_QUALITY, MJPEG_FRAME_BOUNDARY, 9991)
	}

	predictionsStrmr := streamer.NewStreamer[Predictions](streamer.BufferSizeFromTotal(FRAMES_BUFFER_SIZE))
	go predictionsStrmr.Run()
	tensorType := interpreter.GetInputTensor(0).Type()
	if tensorType == tflite.UInt8 {
		go processFramesUint8(analyticsStrmr, predictionsStrmr)
	} else if tensorType == tflite.Float32 {
		go processFramesFloat32(analyticsStrmr, predictionsStrmr)
	} else {
		log.Fatal("Input tensor type", tensorType, "is not supperted")
	}

	http.HandleFunc("/ws", serveInferenceResultWSRequest(predictionsStrmr))
	http.Handle("/", http.FileServer(http.Dir("./public")))

	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	predictionsStrmr.Stop()
	visualizationStrmr.Stop()
	analyticsStrmr.Stop()
	model.Delete()

	if err := server.Shutdown(context.Background()); err != nil {
		log.Fatalf("HTTP shutdown error: %v", err)
	}
}
