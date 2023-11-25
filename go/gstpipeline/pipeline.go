package gstpipeline

import (
	"log"
	"os/exec"
	"strings"
)

func LauchImx219CsiCameraMjpegStream(index uint, width uint, height uint, rWidth uint, rHeight uint, quality uint, boundary string, port uint) {
	cmdSetup := exec.Command(
		"bash", "-c", CsiCameraSetup(IMX219, index, width, height),
	)
	if err := cmdSetup.Run(); err != nil {
		log.Fatal("Cannot setup CsiCamera: ", err)
	}
	log.Print(strings.Join(cmdSetup.Args, " "))
	cmd := exec.Command(
		"bash", "-c", GStreamerLaunch()+
			CsiCameraSource(IMX219, index, width, height)+
			DecodeBin()+
			Rescale(rWidth, rHeight)+
			JpegEncode(quality)+
			MjpegTcpStreamLocalhost(boundary, port),
	)
	log.Print(strings.Join(cmd.Args, " "))
	if err := cmd.Run(); err != nil {
		log.Fatal("Cannot start GStreamer pipeline: ", err)
	}
}

func LauchImx219CsiCameraRgbaStream(index uint, width uint, height uint, rWidth uint, rHeight uint, port uint) {
	cmdSetup := exec.Command(
		"bash", "-c", CsiCameraSetup(IMX219, index, width, height),
	)
	if err := cmdSetup.Run(); err != nil {
		log.Fatal("Cannot setup CsiCamera: ", err)
	}
	log.Print(strings.Join(cmdSetup.Args, " "))
	cmd := exec.Command(
		"bash", "-c", GStreamerLaunch()+
			CsiCameraSource(IMX219, index, width, height)+
			DecodeBin()+
			Rescale(rWidth, rHeight)+
			VideoConvertRgba()+
			TcpStreamLocalhost(port),
	)
	log.Print(strings.Join(cmd.Args, " "))
	if err := cmd.Run(); err != nil {
		log.Fatal("Cannot start GStreamer pipeline: ", err)
	}
}
