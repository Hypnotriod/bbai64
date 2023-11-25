package main

import (
	"bufio"
	"image"
	"io"
	"log"
	"net"
	"net/http"
)

const SERVER_ADDRESS = ":1337"
const BUFFERED_FRAMES_COUNT = 30

func serveTcpRgbaStreamSocket(fWidth int, fHeight int, address string) {
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
		serveTcpRgbaStreamSocketConnection(conn, fWidth, fHeight, address)
		conn.Close()
	}
}

func serveTcpRgbaStreamSocketConnection(conn net.Conn, fWidth int, fHeight int, address string) {
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
		//mux.Broadcast <- chunk
	}
}

func makeStereoCameraMuxer(inputAddr1 string) {
	go serveTcpRgbaStreamSocket(640, 480, inputAddr1)
}

func main() {
	makeStereoCameraMuxer(":9990")

	http.Handle("/", http.FileServer(http.Dir("./public")))

	http.ListenAndServe(SERVER_ADDRESS, nil)
}
