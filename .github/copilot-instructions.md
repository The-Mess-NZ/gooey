# GUIPunk Project Guidelines

## Architecture
- This project is a framebuffer-based GUI service for embedded Linux on Raspberry Pi hardware.
- Keep rendering logic in `pkg/render`, touch input handling in `pkg/input`, IPC in `pkg/ipc`, and UI contracts in `pkg/components`.
- Treat the renderer as the authority for scene updates: mutate component state, then trigger a redraw through the engine.

## Code Style
- Follow the existing Go style in the repo: small focused types, clear exported interfaces at package boundaries, and straightforward control flow.
- Keep drawing inside `Component.Draw`; avoid mixing rendering code into input or IPC handlers.
- Preserve the non-blocking redraw pattern in the renderer so repeated state changes collapse into a single pending frame.
- Long-running loops should accept `context.Context` and stop cleanly when canceled.
- Close hardware and socket resources with `defer` where practical.

## Input and IPC Conventions
- Touch input is calibrated for evdev devices; keep coordinate transforms, axis swapping, and inversion in the input package.
- IPC between GUIPunk and the host uses newline-delimited JSON over a Unix domain socket.
- The IPC server is designed around a single active client connection; new connections may replace older ones.

## Build and Runtime
- Build the main service with `go build -o guipunk cmd/guipunk/main.go`.
- `fbtest.go` is a framebuffer diagnostic utility, not part of the main runtime path.
- Hardware paths such as `/dev/fb0` and `/dev/input/event0` are embedded defaults; keep any changes consistent with the comments in `cmd/guipunk/main.go`.
