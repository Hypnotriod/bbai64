package gstpipeline

import (
	"log"
	"os/exec"
	"strings"
)

func LauchUsbJpegCameraMjpegStream(index uint, width uint, height uint, quality uint, boundary string, port uint) {
	cmd := exec.Command(
		"bash", "-c", GStreamerLaunch()+
			UsbJpegCameraV4l2Source(index)+
			UsbJpegCameraConfig(width, height)+
			JpegDecode()+
			JpegEncode(quality)+
			MjpegTcpStreamLocalhost(boundary, port),
	)
	log.Print(strings.Join(cmd.Args, " "))
	if err := cmd.Run(); err != nil {
		log.Fatal("Cannot start GStreamer pipeline: ", err)
	}
}

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
			TiOvxMultiscaler(rWidth, rHeight)+
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
			VideoScale(rWidth, rHeight)+
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
			VideoScale(rWidth, rHeight)+
			VideoConvertRgb16()+
			TcpStreamLocalhost(port),
	)
	log.Print(strings.Join(cmd.Args, " "))
	if err := cmd.Run(); err != nil {
		log.Fatal("Cannot start GStreamer pipeline: ", err)
	}
}

func LauchImx219CsiCameraBgrStream(index uint, width uint, height uint, rWidth uint, rHeight uint, port uint) {
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
			VideoScale(rWidth, rHeight)+
			VideoConvertBgr()+
			TcpStreamLocalhost(port),
	)
	log.Print(strings.Join(cmd.Args, " "))
	if err := cmd.Run(); err != nil {
		log.Fatal("Cannot start GStreamer pipeline: ", err)
	}
}

func LauchImx219CsiCameraAnalyticsRgbStream1VisualizationMjpegStream2(
	index uint,
	width uint, height uint,
	r1Width uint, r1Height uint,
	boxWidth uint, boxHeight uint,
	port1 uint,
	r2Width uint, r2Height uint,
	quality uint,
	boundary string,
	port2 uint) {
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
			TiOvxMultiscaler(r2Width, r2Height)+
			TiOvxMultiscalerSplit2(
				r1Width, r1Height,
				TiOvxDlColorConvertRgb()+
					VideoBox((r1Width-boxWidth)/2, (r1Width-boxWidth)/2, (r1Height-boxHeight)/2, (r1Height-boxHeight)/2)+
					TcpStreamLocalhost(port1),
				r2Width, r2Height,
				JpegEncode(quality)+
					MjpegTcpStreamLocalhost(boundary, port2),
			),
	)
	log.Print(strings.Join(cmd.Args, " "))
	if err := cmd.Run(); err != nil {
		log.Fatal("Cannot start GStreamer pipeline: ", err)
	}
}

func LauchUsbJpegCameraAnalyticsRgbStream1VisualizationMjpegStream2(
	index uint,
	width uint, height uint,
	r1Width uint, r1Height uint,
	boxWidth uint, boxHeight uint,
	port1 uint,
	r2Width uint, r2Height uint,
	quality uint,
	boundary string,
	port2 uint) {
	cmd := exec.Command(
		"bash", "-c", GStreamerLaunch()+
			UsbJpegCameraV4l2Source(index)+
			UsbJpegCameraConfig(width, height)+
			JpegDecode()+
			TiOvxMultiscaler(r2Width, r2Height)+
			TiOvxMultiscalerSplit2(
				r1Width, r1Height,
				TiOvxDlColorConvertRgb()+
					VideoBox((r1Width-boxWidth)/2, (r1Width-boxWidth)/2, (r1Height-boxHeight)/2, (r1Height-boxHeight)/2)+
					TcpStreamLocalhost(port1),
				r2Width, r2Height,
				JpegEncode(quality)+
					MjpegTcpStreamLocalhost(boundary, port2),
			),
	)
	log.Print(strings.Join(cmd.Args, " "))
	if err := cmd.Run(); err != nil {
		log.Fatal("Cannot start GStreamer pipeline: ", err)
	}
}
