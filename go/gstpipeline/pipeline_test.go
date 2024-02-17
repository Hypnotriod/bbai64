package gstpipeline

import (
	"fmt"
	"testing"
)

func TestBuildAnalyticsPipeline(t *testing.T) {
	fmt.Println(GStreamerLaunch() +
		CsiCameraV4l2Source(0) +
		CsiCameraConfig(0, IMX219, 1920, 1080) +
		TiOvxMultiscaler(1280, 720) +
		TiOvxMultiscalerSplit2(
			320, 180,
			TiOvxDlColorConvertRgb()+
				VideoBox(192/2, 192/2, 52/2, 52/2)+
				TcpStreamLocalhost(9998),
			1280, 720,
			JpegEncode(50)+
				MjpegTcpStreamLocalhost("boundary", 9999),
		))
}
