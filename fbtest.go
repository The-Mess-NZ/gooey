package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"time"

	"github.com/gonutz/framebuffer"
	"github.com/llgcode/draw2d/draw2dimg"
)

type FrameRequest struct {
	fb   *framebuffer.Device
	fbnd image.Rectangle
	im   *image.RGBA
}

func drawFrame(done chan bool, fr FrameRequest) {
	draw.Draw(fr.fb, fr.fbnd, fr.im, image.Pt(0, 0), draw.Src)

	done <- true
}

func main() {
	fb, err := framebuffer.Open("/dev/fb0")

	if err != nil {
		panic(err)
	}

	defer fb.Close()

	// var r uint8 = 0
	// var g uint8 = 0
	// var b uint8 = 0

	// timestamp := time.Now().UnixMilli()

	// var ts float64 = 255.0

	slp := time.Duration(40000000)

	fbnd := fb.Bounds()

	// Initialize the graphic context on an RGBA image
	dest := image.NewRGBA(image.Rect(0, 0, 297, 210.0))
	gc := draw2dimg.NewGraphicContext(dest)

	// Set some properties
	gc.SetFillColor(color.RGBA{0x44, 0xff, 0xff, 0xff})
	gc.SetStrokeColor(color.RGBA{0x44, 0x44, 0x44, 0xff})
	gc.SetLineWidth(5)

	// Draw a closed shape
	gc.MoveTo(10, 10) // should always be called first for a new path
	gc.LineTo(280, 150)
	gc.QuadCurveTo(100, 10, 10, 10)
	gc.Close()
	gc.FillStroke()

	fbdone := make(chan bool, 1)

	for {
		time.Sleep(slp)
		request := FrameRequest{
			fb:   fb,
			fbnd: fbnd,
			im:   dest,
		}
		drawFrame(fbdone, request)
		<-fbdone
		fmt.Println("we done brah")
	}
}
