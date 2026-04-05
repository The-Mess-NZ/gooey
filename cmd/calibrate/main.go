package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/The-Mess-NZ/gui-punk/pkg/components"
	"github.com/The-Mess-NZ/gui-punk/pkg/input"
	"github.com/The-Mess-NZ/gui-punk/pkg/render"
	"github.com/llgcode/draw2d/draw2dimg"
)

const fbDevicePath = "/dev/fb0"
const configFileName = "config.json"

type calibrationStep struct {
	key         string
	label       string
	instruction string
	target      image.Point
}

type rawPoint struct {
	x int32
	y int32
}

type crosshairComponent struct {
	id     string
	center image.Point
	size   int
	color  color.Color
}

type rawTouchCollector struct {
	samples chan rawPoint
}

func main() {
	configPath, err := ensureConfigLoaded()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	touchCfg := components.TouchSettings()
	w := touchCfg.ScreenXPixels
	h := touchCfg.ScreenYPixels

	if touchCfg.IsLandscape {
		w, h = max(w, h), min(w, h)
	} else {
		w, h = min(w, h), max(w, h)
	}

	viewport := image.Rect(0, 0, w, h)

	steps := []calibrationStep{
		{key: "top_left", label: "1/4", instruction: "Tap the top-left crosshair", target: image.Pt(24, 24)},
		{key: "top_right", label: "2/4", instruction: "Tap the top-right crosshair", target: image.Pt(viewport.Max.X-24, 24)},
		{key: "bottom_left", label: "3/4", instruction: "Tap the bottom-left crosshair", target: image.Pt(24, viewport.Max.Y-24)},
		{key: "bottom_right", label: "4/4", instruction: "Tap the bottom-right crosshair", target: image.Pt(viewport.Max.X-24, viewport.Max.Y-24)},
	}

	engine, err := render.NewEngine(fbDevicePath)
	if err != nil {
		log.Fatalf("Failed to initialize Render Engine on %s: %v", fbDevicePath, err)
	}
	defer engine.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go engine.Loop(ctx)

	collector := &rawTouchCollector{samples: make(chan rawPoint, 16)}
	listener, err := input.NewRawTouchListener(touchCfg.DevicePath, collector)
	if err != nil {
		log.Fatalf("Failed to open raw touch device on %s: %v", touchCfg.DevicePath, err)
	}
	go listener.Start(ctx)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	measurements := make(map[string]rawPoint, len(steps))
	currentStep := 0
	renderStep(engine, viewport, steps[currentStep], false, "Waiting for release event")
	log.Printf("Calibration started using %s; results will be written to %s", touchCfg.DevicePath, configPath)

	for {
		select {
		case sample := <-collector.samples:
			step := steps[currentStep]
			measurements[step.key] = sample
			log.Printf("Captured %s raw sample: x=%d y=%d", step.key, sample.x, sample.y)
			currentStep++
			if currentStep >= len(steps) {
				updated := calculateCalibration(measurements, touchCfg)
				components.SetTouchSettings(updated)
				if err := components.SaveConfig(configPath); err != nil {
					log.Fatalf("Failed to save updated config: %v", err)
				}
				renderResult(engine, viewport, updated, configPath)
				log.Printf("Calibration complete: minX=%d maxX=%d minY=%d maxY=%d swapXY=%t invertX=%t invertY=%t", updated.MinXRaw, updated.MaxXRaw, updated.MinYRaw, updated.MaxYRaw, updated.IsLandscape, updated.InvertX, updated.InvertY)
				log.Println("Calibration saved. Press Ctrl+C to exit.")

				<-stop
				return
			}
			renderStep(engine, viewport, steps[currentStep], false, fmt.Sprintf("Last raw sample: %d, %d", sample.x, sample.y))

		case <-stop:
			log.Println("Calibration cancelled")
			return
		}
	}
}

func ensureConfigLoaded() (string, error) {
	configPath, err := components.LoadConfigFromCandidates(
		os.Getenv("GUIPUNK_CONFIG_PATH"),
		filepath.Join(".", configFileName),
		filepath.Join("..", configFileName),
		filepath.Join("..", "..", configFileName),
		filepath.Join("/etc", "guipunk", configFileName),
	)
	if err != nil {
		return "", err
	}
	if configPath != "" {
		return configPath, nil
	}
	defaultPath := filepath.Join(".", configFileName)
	if err := components.SaveConfig(defaultPath); err != nil {
		return "", err
	}
	if err := components.LoadConfig(defaultPath); err != nil {
		return "", err
	}
	return defaultPath, nil
}

func (c *crosshairComponent) ID() string {
	return c.id
}

func (c *crosshairComponent) Draw(gc *draw2dimg.GraphicContext) {
	gc.SetStrokeColor(c.color)
	gc.SetLineWidth(2)
	gc.BeginPath()
	gc.MoveTo(float64(c.center.X-c.size), float64(c.center.Y))
	gc.LineTo(float64(c.center.X+c.size), float64(c.center.Y))
	gc.MoveTo(float64(c.center.X), float64(c.center.Y-c.size))
	gc.LineTo(float64(c.center.X), float64(c.center.Y+c.size))
	gc.Stroke()
	gc.BeginPath()
	gc.ArcTo(float64(c.center.X), float64(c.center.Y), 8, 8, 0, 6.28318)
	gc.Stroke()
}

func (c *crosshairComponent) BoundingBox() image.Rectangle {
	return image.Rect(c.center.X-c.size, c.center.Y-c.size, c.center.X+c.size, c.center.Y+c.size)
}

func (c *crosshairComponent) HandleTouch(x, y int, isRelease bool) components.TouchResult {
	return components.TouchResult{}
}

func (c *rawTouchCollector) HandleRawTouch(rawX, rawY int32, isRelease bool) {
	if !isRelease {
		return
	}
	select {
	case c.samples <- rawPoint{x: rawX, y: rawY}:
	default:
	}
}

func renderStep(engine *render.Engine, viewport image.Rectangle, step calibrationStep, complete bool, footer string) {
	comps, err := buildScene(viewport, step.label, step.instruction, footer)
	if err != nil {
		log.Printf("Failed to build calibration scene: %v", err)
		return
	}
	colorValue := color.RGBA{0xF2, 0xE9, 0x76, 0xFF}
	if complete {
		colorValue = color.RGBA{0x95, 0xE0, 0x91, 0xFF}
	}
	comps = append(comps, &crosshairComponent{id: "crosshair", center: step.target, size: 14, color: colorValue})
	engine.SetComponents(comps)
}

func renderResult(engine *render.Engine, viewport image.Rectangle, touchCfg components.TouchConfig, configPath string) {
	message := fmt.Sprintf("Saved to %s", configPath)
	footer := fmt.Sprintf("swapXY=%t invertX=%t invertY=%t", touchCfg.IsLandscape, touchCfg.InvertX, touchCfg.InvertY)
	comps, err := buildScene(viewport, "Done", message, footer)
	if err != nil {
		log.Printf("Failed to build calibration result scene: %v", err)
		return
	}
	engine.SetComponents(comps)
}

func buildScene(viewport image.Rectangle, title, message, footer string) ([]components.Component, error) {
	doc := components.SceneDocument{
		Version: components.SceneVersionAlpha1,
		Root: components.SceneNode{
			ID:   "calibration-root",
			Type: components.NodeTypeContainer,
			Layout: &components.Layout{
				Direction: components.LayoutDirectionVertical,
				Gap:       8,
				Padding:   components.Insets{Top: 10, Right: 10, Bottom: 10, Left: 10},
			},
			Style: &components.Style{Background: "#10161BFF"},
			Children: []components.SceneNode{
				{
					ID:     "title",
					Type:   components.NodeTypeLabel,
					Text:   title,
					Bounds: &components.Rect{Height: 24},
					Style:  &components.Style{Background: "#1B2732FF", Foreground: "#F7F1D5", FontSize: 15, TextPadding: 4},
				},
				{
					ID:     "message",
					Type:   components.NodeTypeLabel,
					Text:   message,
					Bounds: &components.Rect{Height: 34},
					Style:  &components.Style{Foreground: "#F2F2E9", FontSize: 13, TextPadding: 4},
				},
				{
					ID:   "spacer",
					Type: components.NodeTypeContainer,
				},
				{
					ID:     "footer",
					Type:   components.NodeTypeLabel,
					Text:   footer,
					Bounds: &components.Rect{Height: 24},
					Style:  &components.Style{Foreground: "#98A8B5", FontSize: 11, TextPadding: 4},
				},
			},
		},
	}

	return components.BuildScene(doc, viewport)
}

func calculateCalibration(samples map[string]rawPoint, base components.TouchConfig) components.TouchConfig {
	topLeft := samples["top_left"]
	topRight := samples["top_right"]
	bottomLeft := samples["bottom_left"]
	bottomRight := samples["bottom_right"]

	noSwapScore := abs32(topRight.x-topLeft.x) + abs32(bottomRight.x-bottomLeft.x) + abs32(bottomLeft.y-topLeft.y) + abs32(bottomRight.y-topRight.y)
	swapScore := abs32(topRight.y-topLeft.y) + abs32(bottomRight.y-bottomLeft.y) + abs32(bottomLeft.x-topLeft.x) + abs32(bottomRight.x-topRight.x)
	swapXY := swapScore > noSwapScore

	leftA, rightA := averagedHorizontal(topLeft, topRight, bottomLeft, bottomRight, swapXY)
	topA, bottomA := averagedVertical(topLeft, topRight, bottomLeft, bottomRight, swapXY)

	base.IsLandscape = swapXY
	base.MinXRaw = min32(leftA, rightA)
	base.MaxXRaw = max32(leftA, rightA)
	base.MinYRaw = min32(topA, bottomA)
	base.MaxYRaw = max32(topA, bottomA)
	base.InvertX = leftA > rightA
	base.InvertY = topA > bottomA
	return base
}

func averagedHorizontal(topLeft, topRight, bottomLeft, bottomRight rawPoint, swapXY bool) (int32, int32) {
	if swapXY {
		return average32(topLeft.y, bottomLeft.y), average32(topRight.y, bottomRight.y)
	}
	return average32(topLeft.x, bottomLeft.x), average32(topRight.x, bottomRight.x)
}

func averagedVertical(topLeft, topRight, bottomLeft, bottomRight rawPoint, swapXY bool) (int32, int32) {
	if swapXY {
		return average32(topLeft.x, topRight.x), average32(bottomLeft.x, bottomRight.x)
	}
	return average32(topLeft.y, topRight.y), average32(bottomLeft.y, bottomRight.y)
}

func average32(a, b int32) int32 {
	return (a + b) / 2
}

func abs32(value int32) int32 {
	if value < 0 {
		return -value
	}
	return value
}

func min32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func max32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
