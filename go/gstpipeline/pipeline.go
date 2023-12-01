package gstpipeline

import (
	"log"
	"os/exec"
	"strings"
)

func LauchImx219CsiCameraMjpegStream(index uint, width uint, height uint, rWidth uint, rHeight uint, quality uint, boundary string, port uint) {
	cmdSetup := exec.Command("bash", "-c", CsiCameraSetup(IMX219, index, width, height))
	log.Print(strings.Join(cmdSetup.Args, " "))
	if err := cmdSetup.Run(); err != nil {
		log.Fatal("Cannot setup CSI Camera: ", err)
	}
	cmd := exec.Command(
		"bash", "-c", GStreamerLaunch()+
			CsiCameraV4l2Source(index)+
			CsiCameraConfig(index, IMX219, width, height)+
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

func LauchImx219CsiStereoCameraMjpegStream(width uint, height uint, rWidth uint, rHeight uint, quality uint, boundary string, port uint) {
	cmdSetupL := exec.Command("bash", "-c", CsiCameraSetup(IMX219, 0, width, height))
	log.Print(strings.Join(cmdSetupL.Args, " "))
	if err := cmdSetupL.Run(); err != nil {
		log.Fatal("Cannot setup left CSI Camera: ", err)
	}
	cmdSetupR := exec.Command("bash", "-c", CsiCameraSetup(IMX219, 1, width, height))
	log.Print(strings.Join(cmdSetupR.Args, " "))
	if err := cmdSetupR.Run(); err != nil {
		log.Fatal("Cannot setup right CSI Camera: ", err)
	}
	cmd := exec.Command(
		"bash", "-c", GStreamerLaunch()+
			GlStereoMix(
				CsiCameraV4l2Source(0),
				CsiCameraV4l2Source(1),
				CsiCameraConfig(0, IMX219, width, height),
				CsiCameraConfig(1, IMX219, width, height),
			)+
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

func LauchImx219CsiCameraRgb16Stream(index uint, width uint, height uint, rWidth uint, rHeight uint, port uint) {
	cmdSetup := exec.Command(
		"bash", "-c", CsiCameraSetup(IMX219, index, width, height),
	)
	if err := cmdSetup.Run(); err != nil {
		log.Fatal("Cannot setup CSI Camera: ", err)
	}
	log.Print(strings.Join(cmdSetup.Args, " "))
	cmd := exec.Command(
		"bash", "-c", GStreamerLaunch()+
			CsiCameraV4l2Source(index)+
			CsiCameraConfig(index, IMX219, width, height)+
			DecodeBin()+
			Rescale(rWidth, rHeight)+
			VideoConvertRgb16()+
			TcpStreamLocalhost(port),
	)
	log.Print(strings.Join(cmd.Args, " "))
	if err := cmd.Run(); err != nil {
		log.Fatal("Cannot start GStreamer pipeline: ", err)
	}
}
