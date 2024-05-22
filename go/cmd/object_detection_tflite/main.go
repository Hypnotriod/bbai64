package main

import (
	"bbai64/gstpipeline"
	"bbai64/streamer"
	"bbai64/titfldelegate"
	"bbai64/vehicle"
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
const RESCALE_ANALYTICS_WIDTH = 640
const RESCALE_ANALYTICS_HEIGHT = 360
const TENSOR_WIDTH = 320
const TENSOR_HEIGHT = 320
const MEAN = 127.5
const SCALE = 1.0 / 127.5
const SCORES_TENSOR_INDEX = 2  // 0
const BOXES_TENSOR_INDEX = 0   // 1
const COUNT_TENSOR_INDEX = 3   // 2
const CLASSES_TENSOR_INDEX = 1 // 3
const CHANNELS_NUM = 3
const TENSOR_SIZE = TENSOR_WIDTH * TENSOR_HEIGHT * CHANNELS_NUM
const FRAME_ADJUST_SCALE = float32(TENSOR_HEIGHT) / float32(RESCALE_ANALYTICS_HEIGHT)
const PREDICT_EACH_FRAME = 1
const MIN_SCORE = 0.6
const USE_DELEGATE = true
const MODEL_PATH = "model/ssdLite-mobDet-DSP-coco-320x320/model/ssdlite_mobiledet_dsp_320x320_coco_20200519.tflite"
const LABELS_PATH = "model/ssdLite-mobDet-DSP-coco-320x320/labels.txt"
const ARTIFACTS_PATH = "model/ssdLite-mobDet-DSP-coco-320x320/artifacts"
const TFL_DELEGATE_PATH = "/usr/lib/libtidl_tfl_delegate.so"

type PixelsRGB []byte

type Chunk struct {
	Data [CHUNK_SIZE]byte
	Size int
}

type Detection struct {
	Label string  `json:"label"`
	Class int     `json:"class"`
	Score float32 `json:"score"`
	Xmin  float32 `json:"xmin"`
	Ymin  float32 `json:"ymin"`
	Xmax  float32 `json:"xmax"`
	Ymax  float32 `json:"ymax"`
}

type Detections []Detection

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

func serveInferenceResultWSRequest(strmr *streamer.Streamer[Detections]) func(w http.ResponseWriter, r *http.Request) {
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
			detections, ok := <-client.C
			if !ok {
				break
			}
			message, _ := json.Marshal(detections)
			conn.SetWriteDeadline(time.Now().Add(CONNECTION_TIMEOUT))
			err = conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Print("Websocket write error: ", err)
				break
			}
		}
		vehicle.Reset()
		log.Print("Websocket connection terminated with ", r.Host)
	}
}

func serveAnalyticsStreamTcpSocket(width int, height int, strmr *streamer.Streamer[PixelsRGB], address string) {
	soc, err := net.Listen("tcp", address)
	defer soc.Close()
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
	defer soc.Close()
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
		if !strmr.IsRunning() {
			break
		}
	}
}

func serveVisualizationMjpegStreamTcpSocketConnection(conn net.Conn, strmr *streamer.Streamer[Chunk], address string) {
	log.Print("Accepted input stream at ", address)
	var buffIndex int32
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

		client := strmr.NewClient(streamer.BufferSizeFromTotal(CHUNKS_BUFFER_SIZE))
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
	strmr := streamer.NewStreamer[Chunk](streamer.BufferSizeFromTotal(CHUNKS_BUFFER_SIZE))
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

func processFramesUint8(frameStrmr *streamer.Streamer[PixelsRGB], detStrmr *streamer.Streamer[Detections]) {
	inputTensor := (*[TENSOR_SIZE]byte)(interpreter.GetInputTensor(0).Data())
	buffer := [FRAMES_BUFFER_SIZE]Detections{}
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
		detStrmr.Broadcast(&buffer[buffIndex])
		buffIndex = (buffIndex + 1) % FRAMES_BUFFER_SIZE
	}
}

func processFramesFloat32(frameStrmr *streamer.Streamer[PixelsRGB], detStrmr *streamer.Streamer[Detections]) {
	inputTensor := (*[TENSOR_SIZE]float32)(interpreter.GetInputTensor(0).Data())
	buffer := [FRAMES_BUFFER_SIZE]Detections{}
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
		detStrmr.Broadcast(&buffer[buffIndex])
		buffIndex = (buffIndex + 1) % FRAMES_BUFFER_SIZE
	}
}

func predict(detections *Detections) {
	startTime := time.Now()
	status := interpreter.Invoke()
	if status != tflite.OK {
		log.Println("Interpreter invoke failed")
		return
	}
	endTime := time.Since(startTime)
	log.Println("---", endTime, "---")
	scores := interpreter.GetOutputTensor(SCORES_TENSOR_INDEX).Float32s()
	boxes := interpreter.GetOutputTensor(BOXES_TENSOR_INDEX).Float32s()
	count := interpreter.GetOutputTensor(COUNT_TENSOR_INDEX).Float32s()
	classes := interpreter.GetOutputTensor(CLASSES_TENSOR_INDEX).Float32s()
	if cap(*detections) == 0 && int(count[0]) > 0 {
		*detections = make(Detections, 0, int(count[0]))
	}
	*detections = (*detections)[:0]
	for n := 0; n < int(count[0]); n++ {
		score := scores[n]
		if score < MIN_SCORE {
			continue
		}
		label := labels[int(classes[n])]
		ymin := boxes[n*4+0]
		xmin := boxes[n*4+1]
		ymax := boxes[n*4+2]
		xmax := boxes[n*4+3]
		fmt.Printf("    %s score: %.2g [x: %d  y: %d w: %d h: %d]\n",
			label,
			score,
			int(xmin*TENSOR_WIDTH),
			int(ymin*TENSOR_HEIGHT),
			int((xmax-xmin)*TENSOR_WIDTH),
			int((ymax-ymin)*TENSOR_HEIGHT),
		)
		*detections = append(*detections, Detection{
			Label: label,
			Class: int(classes[n]),
			Score: score,
			Xmin:  FRAME_ADJUST_SCALE*(xmin-0.5) + 0.5,
			Ymin:  FRAME_ADJUST_SCALE*(ymin-0.5) + 0.5,
			Xmax:  FRAME_ADJUST_SCALE*(xmax-0.5) + 0.5,
			Ymax:  FRAME_ADJUST_SCALE*(ymax-0.5) + 0.5,
		})
	}
}

func main() {
	server := &http.Server{Addr: SERVER_ADDRESS}

	model := initModel()
	defer model.Delete()

	strmrAnalytics := makeAnalyticsCameraStreamer(":9990", "/mjpeg_stream1")
	defer strmrAnalytics.Stop()

	strmrVisualization := makeVisualizationMjpegStreamer(":9991", "/mjpeg_stream2")
	defer strmrVisualization.Stop()

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

	detStrmr := streamer.NewStreamer[Detections](streamer.BufferSizeFromTotal(FRAMES_BUFFER_SIZE))
	go detStrmr.Run()
	defer detStrmr.Stop()

	tensorType := interpreter.GetInputTensor(0).Type()
	if tensorType == tflite.UInt8 {
		go processFramesUint8(strmrAnalytics, detStrmr)
	} else if tensorType == tflite.Float32 {
		go processFramesFloat32(strmrAnalytics, detStrmr)
	} else {
		log.Fatal("Input tensor type", tensorType, "is not supperted")
	}

	http.HandleFunc("/ws", serveInferenceResultWSRequest(detStrmr))
	http.Handle("/", http.FileServer(http.Dir("./public")))

	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	if err := server.Shutdown(context.Background()); err != nil {
		log.Fatalf("HTTP shutdown error: %v", err)
	}
}
