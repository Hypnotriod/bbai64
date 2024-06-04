package main

import (
	"bbai64/gstpipeline"
	"bbai64/streamer"
	"bbai64/titfldelegate"
	"bufio"
	"context"
	"encoding/hex"
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
const RESCALE_ANALYTICS_WIDTH = 960
const RESCALE_ANALYTICS_HEIGHT = 540
const TENSOR_WIDTH = 512
const TENSOR_HEIGHT = 512
const MEAN = 0
const SCALE = 1.0 / 255.0
const CHANNELS_NUM = 3
const TENSOR_SIZE = TENSOR_WIDTH * TENSOR_HEIGHT * CHANNELS_NUM
const PREDICT_EACH_FRAME = 1
const USE_DELEGATE = true
const MODEL_PATH = "model/ssLite-deeplabv3_mobv2-ade20k32-mlperf-512x512/model/deeplabv3_mnv2_ade20k32_float.tflite"
const LABELS_PATH = "model/ssLite-deeplabv3_mobv2-ade20k32-mlperf-512x512/labels.txt"
const COLORS_PATH = "model/ssLite-deeplabv3_mobv2-ade20k32-mlperf-512x512/colors.txt"
const ARTIFACTS_PATH = "model/ssLite-deeplabv3_mobv2-ade20k32-mlperf-512x512/artifacts"
const TFL_DELEGATE_PATH = "/usr/lib/libtidl_tfl_delegate.so"

type PixelsRGB []byte

type Chunk struct {
	Data [CHUNK_SIZE]byte
	Size int
}

type InputTensor interface {
	*[TENSOR_SIZE]byte | *[TENSOR_SIZE]float32
}

var interpreter *tflite.Interpreter
var labels []string
var colors [][]byte

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

func handleSegmentationMjpegStreamRequest(width int, height int, strmr *streamer.Streamer[PixelsRGB]) func(w http.ResponseWriter, req *http.Request) {
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
	strmr := streamer.NewStreamer[Chunk](CHUNKS_BUFFER_SIZE/2 - 2).Run()
	go serveVisualizationMjpegStreamTcpSocket(strmr, inputAddr)
	http.HandleFunc(outputAddr, handleVisualizationMjpegStreamRequest(strmr))
	return strmr
}

func makeAnalyticsCameraStreamer(inputAddr string) *streamer.Streamer[PixelsRGB] {
	strmr := streamer.NewStreamer[PixelsRGB](streamer.BufferSizeFromTotal(FRAMES_BUFFER_SIZE)).Run()
	go serveAnalyticsStreamTcpSocket(TENSOR_WIDTH, TENSOR_HEIGHT, strmr, inputAddr)
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
	labels = strings.Split(strings.Trim(string(labelsRaw), "\r"), "\n")

	colorsRaw, err := os.ReadFile(COLORS_PATH)
	if err != nil {
		log.Fatal("Cannot read model labels: ", err)
	}
	colorsStr := strings.Split(strings.Trim(string(colorsRaw), "\r"), "\n")
	for _, c := range colorsStr {
		c = strings.Trim(c, "#")
		crs, err := hex.DecodeString(c)
		if err != nil {
			log.Fatal("Cannot parse colors: ", err)
		}
		colors = append(colors, crs)
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

func processFrames[T InputTensor](inputTensor T, frameStrmr *streamer.Streamer[PixelsRGB], segmStrmr *streamer.Streamer[PixelsRGB]) {
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
		switch t := any(inputTensor).(type) {
		case *[TENSOR_SIZE]byte:
			copy(t[:], *frame)
		case *[TENSOR_SIZE]float32:
			for i, b := range *frame {
				t[i] = (float32(b) - MEAN) * SCALE
			}
		}
		predict(frame)
		if !segmStrmr.Broadcast(frame) {
			break
		}
		buffIndex = (buffIndex + 1) % FRAMES_BUFFER_SIZE
	}
}

func predict(frame *PixelsRGB) {
	startTime := time.Now()
	status := interpreter.Invoke()
	if status != tflite.OK {
		log.Println("Interpreter invoke failed")
		return
	}
	endTime := time.Since(startTime)
	result := interpreter.GetOutputTensor(0).UInt8s()
	n := 0
	for _, class := range result {
		(*frame)[n+0] = colors[class][0]
		(*frame)[n+1] = colors[class][1]
		(*frame)[n+2] = colors[class][2]
		n += CHANNELS_NUM
	}
	fmt.Println("Time taken", endTime)
}

func runServer(server *http.Server) {
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("HTTP server error: %v", err)
	}
}

func main() {
	server := &http.Server{Addr: SERVER_ADDRESS}
	model := initModel()
	analyticsStrmr := makeAnalyticsCameraStreamer(":9990")
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

	segmentationStrmr := streamer.NewStreamer[PixelsRGB](streamer.BufferSizeFromTotal(FRAMES_BUFFER_SIZE)).Run()
	http.HandleFunc("/mjpeg_stream1", handleSegmentationMjpegStreamRequest(TENSOR_WIDTH, TENSOR_HEIGHT, segmentationStrmr))

	tensorType := interpreter.GetInputTensor(0).Type()
	if tensorType == tflite.UInt8 {
		inputTensor := (*[TENSOR_SIZE]byte)(interpreter.GetInputTensor(0).Data())
		go processFrames(inputTensor, analyticsStrmr, segmentationStrmr)
	} else if tensorType == tflite.Float32 {
		inputTensor := (*[TENSOR_SIZE]float32)(interpreter.GetInputTensor(0).Data())
		go processFrames(inputTensor, analyticsStrmr, segmentationStrmr)
	} else {
		log.Fatal("Input tensor type", tensorType, "is not supported")
	}

	http.Handle("/", http.FileServer(http.Dir("./public")))

	go runServer(server)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	segmentationStrmr.Stop()
	visualizationStrmr.Stop()
	analyticsStrmr.Stop()
	model.Delete()

	if err := server.Shutdown(context.Background()); err != nil {
		log.Fatalf("HTTP shutdown error: %v", err)
	}
}
