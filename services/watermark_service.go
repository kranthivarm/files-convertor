package services

import (
	"context"
	"fmt"
	"image"
	_ "image/png"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// WatermarkRegion represents a detected watermark bounding box.
type WatermarkRegion struct {
	X, Y, W, H int
	Score       float64
}

// AutoRemoveWatermark detects and removes watermarks from a video automatically.
func AutoRemoveWatermark(inputPath, outputPath string) error {
	// Step 1: Get original video dimensions
	origW, origH, err := getVideoDimensions(inputPath)
	if err != nil {
		return fmt.Errorf("could not read video dimensions: %w", err)
	}
	log.Printf("[watermark] original video: %dx%d", origW, origH)

	// Step 2: Extract sample frames (downscaled for fast analysis)
	tmpDir, err := os.MkdirTemp("", "wm-frames-*")
	if err != nil {
		return fmt.Errorf("could not create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	const analysisWidth = 640
	const frameCount = 15

	framePaths, err := extractFrames(inputPath, tmpDir, analysisWidth, frameCount)
	if err != nil {
		return fmt.Errorf("frame extraction failed: %w", err)
	}
	log.Printf("[watermark] extracted %d frames for analysis", len(framePaths))

	if len(framePaths) < 2 {
		return fmt.Errorf("video too short — extracted only %d frames, need at least 2 for analysis", len(framePaths))
	}

	// Step 3: Load frames into memory
	frames, err := loadFrames(framePaths)
	if err != nil {
		return fmt.Errorf("could not load extracted frames: %w", err)
	}

	// Step 4: Detect watermark regions
	regions, err := detectAllWatermarkRegions(frames)
	if err != nil {
		return err
	}

	// Step 5: Scale coordinates back to original resolution and build delogo chain
	analysisW := frames[0].Bounds().Dx()
	analysisH := frames[0].Bounds().Dy()
	scaleX := float64(origW) / float64(analysisW)
	scaleY := float64(origH) / float64(analysisH)

	var scaledRegions []WatermarkRegion
	for _, r := range regions {
		x := int(float64(r.X) * scaleX)
		y := int(float64(r.Y) * scaleY)
		w := int(float64(r.W) * scaleX)
		h := int(float64(r.H) * scaleY)

		// Add padding for cleaner blending
		pad := int(math.Max(float64(w), float64(h))*0.12) + 3
		x = max(0, x-pad)
		y = max(0, y-pad)
		w = min(origW-x, w+2*pad)
		h = min(origH-y, h+2*pad)

		log.Printf("[watermark] region → x=%d y=%d w=%d h=%d (score=%.2f)", x, y, w, h, r.Score)
		scaledRegions = append(scaledRegions, WatermarkRegion{X: x, Y: y, W: w, H: h})
	}

	// Step 6: Apply chained delogo filters
	return applyMultiDelogo(inputPath, outputPath, scaledRegions)
}

// ──────────────────────────────────────────────────────────
//  FFmpeg / FFprobe helpers
// ──────────────────────────────────────────────────────────

func getVideoDimensions(videoPath string) (int, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=p=0:s=x",
		videoPath,
	)

	out, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("ffprobe failed: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(string(out)), "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected ffprobe output: %q", strings.TrimSpace(string(out)))
	}

	w, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid width: %w", err)
	}
	h, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid height: %w", err)
	}

	return w, h, nil
}

func extractFrames(videoPath, tmpDir string, targetWidth, count int) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pattern := filepath.Join(tmpDir, "frame_%03d.png")

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", videoPath,
		"-vf", fmt.Sprintf("fps=1,scale=%d:-2", targetWidth),
		"-frames:v", strconv.Itoa(count),
		"-y",
		pattern,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("%v — %s", err, string(output))
	}

	var paths []string
	for i := 1; i <= count; i++ {
		p := filepath.Join(tmpDir, fmt.Sprintf("frame_%03d.png", i))
		if _, err := os.Stat(p); err == nil {
			paths = append(paths, p)
		}
	}

	return paths, nil
}

func loadFrames(paths []string) ([]image.Image, error) {
	var frames []image.Image
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		img, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			continue
		}
		frames = append(frames, img)
	}
	if len(frames) < 2 {
		return nil, fmt.Errorf("could not decode enough frames (got %d, need ≥ 2)", len(frames))
	}
	return frames, nil
}

// applyMultiDelogo chains multiple delogo filters in a single FFmpeg call.
func applyMultiDelogo(inputPath, outputPath string, regions []WatermarkRegion) error {
	if len(regions) == 0 {
		return fmt.Errorf("no watermark regions to remove")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Build chained filter: "delogo=...,delogo=...,delogo=..."
	var filters []string
	for _, r := range regions {
		filters = append(filters, fmt.Sprintf("delogo=x=%d:y=%d:w=%d:h=%d", r.X, r.Y, r.W, r.H))
	}
	filterChain := strings.Join(filters, ",")
	log.Printf("[watermark] applying filter: %s", filterChain)

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", inputPath,
		"-vf", filterChain,
		"-c:a", "copy",
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("ffmpeg timed out after 5 minutes")
	}
	if err != nil {
		return fmt.Errorf("ffmpeg delogo failed: %v — %s", err, string(output))
	}

	return nil
}

// ──────────────────────────────────────────────────────────
//  Watermark detection — dual strategy
// ──────────────────────────────────────────────────────────

// detectAllWatermarkRegions uses two strategies and merges results:
//  1. Variance-based detection (good for videos with motion)
//  2. Corner-scan edge detection (good for static/AI-generated content)
func detectAllWatermarkRegions(frames []image.Image) ([]WatermarkRegion, error) {
	bounds := frames[0].Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	log.Printf("[watermark] analysis frame size: %dx%d, %d frames loaded", width, height, len(frames))

	// Strategy 1: Variance-based detection
	varianceRegions := detectByVariance(frames, width, height, bounds)
	log.Printf("[watermark] variance strategy found %d candidates", len(varianceRegions))

	// Strategy 2: Corner-scan edge-based detection
	cornerRegions := detectByCornerScan(frames[0], width, height, bounds)
	log.Printf("[watermark] corner-scan strategy found %d candidates", len(cornerRegions))

	// Merge results, remove duplicates (overlapping regions)
	all := append(varianceRegions, cornerRegions...)

	if len(all) == 0 {
		return nil, fmt.Errorf(
			"no watermark detected — the video may not contain a visible static watermark, " +
				"or the watermark blends too closely with the content")
	}

	// Deduplicate overlapping regions — keep higher-scoring one
	merged := deduplicateRegions(all)
	log.Printf("[watermark] after dedup: %d regions to remove", len(merged))

	// Cap at 5 regions max
	if len(merged) > 5 {
		merged = merged[:5]
	}

	return merged, nil
}

// ── Strategy 1: Variance-based (for videos with motion) ──

func detectByVariance(frames []image.Image, width, height int, bounds image.Rectangle) []WatermarkRegion {
	totalPixels := width * height

	// Compute per-pixel max color range across all frames
	rangeMap := make([][]uint32, height)
	for py := 0; py < height; py++ {
		rangeMap[py] = make([]uint32, width)
		for px := 0; px < width; px++ {
			var minR, minG, minB uint32 = 255, 255, 255
			var maxR, maxG, maxB uint32 = 0, 0, 0

			for _, fr := range frames {
				r, g, b, _ := fr.At(px+bounds.Min.X, py+bounds.Min.Y).RGBA()
				r8, g8, b8 := r>>8, g>>8, b>>8
				if r8 < minR { minR = r8 }
				if r8 > maxR { maxR = r8 }
				if g8 < minG { minG = g8 }
				if g8 > maxG { maxG = g8 }
				if b8 < minB { minB = b8 }
				if b8 > maxB { maxB = b8 }
			}

			maxRange := maxR - minR
			if d := maxG - minG; d > maxRange { maxRange = d }
			if d := maxB - minB; d > maxRange { maxRange = d }
			rangeMap[py][px] = maxRange
		}
	}

	// Adaptive threshold via median
	allRanges := make([]uint32, 0, totalPixels)
	for py := 0; py < height; py++ {
		for px := 0; px < width; px++ {
			allRanges = append(allRanges, rangeMap[py][px])
		}
	}
	sort.Slice(allRanges, func(i, j int) bool { return allRanges[i] < allRanges[j] })
	medianRange := allRanges[len(allRanges)/2]

	log.Printf("[watermark] variance: median pixel range = %d", medianRange)

	if medianRange < 5 {
		log.Printf("[watermark] variance: video too static (median=%d), skipping variance strategy", medianRange)
		return nil
	}

	threshold := uint32(float64(medianRange) * 0.25)
	if threshold < 8 { threshold = 8 }
	if threshold > 25 { threshold = 25 }

	// Build static mask
	staticMask := make([][]bool, height)
	totalStatic := 0
	for py := 0; py < height; py++ {
		staticMask[py] = make([]bool, width)
		for px := 0; px < width; px++ {
			if rangeMap[py][px] < threshold {
				staticMask[py][px] = true
				totalStatic++
			}
		}
	}

	staticRatio := float64(totalStatic) / float64(totalPixels)
	log.Printf("[watermark] variance: %.1f%% of pixels are static (threshold=%d)", staticRatio*100, threshold)

	if staticRatio > 0.90 {
		log.Printf("[watermark] variance: too many static pixels, skipping")
		return nil
	}

	// Find connected components
	visited := make([][]bool, height)
	for i := range visited { visited[i] = make([]bool, width) }

	var components []wmComponent
	for py := 0; py < height; py++ {
		for px := 0; px < width; px++ {
			if staticMask[py][px] && !visited[py][px] {
				comp := wmFloodFill(staticMask, visited, px, py, width, height)
				components = append(components, comp)
			}
		}
	}

	// Filter: 0.02% to 10% of frame
	minSize := totalPixels / 5000
	maxSize := totalPixels * 10 / 100

	var results []WatermarkRegion
	for _, c := range components {
		if c.size >= minSize && c.size <= maxSize {
			score := wmScoreCandidate(c, width, height)
			if score > 1.5 {
				results = append(results, WatermarkRegion{
					X: c.minX, Y: c.minY,
					W: c.maxX - c.minX + 1,
					H: c.maxY - c.minY + 1,
					Score: score,
				})
			}
		}
	}

	// Sort by score descending, keep top 3
	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if len(results) > 3 {
		results = results[:3]
	}

	return results
}

// ── Strategy 2: Corner-scan edge detection (for static content) ──

// detectByCornerScan checks common watermark positions for text-like patterns.
// Works even when the entire video is static (AI-generated content).
func detectByCornerScan(frame image.Image, width, height int, bounds image.Rectangle) []WatermarkRegion {
	// Define scan zones: corners and edge strips where watermarks typically appear
	type scanZone struct {
		name       string
		x, y, w, h int
		priority   float64
	}

	// Watermark regions as percentage of frame, then converted to pixels
	cornerW := max(80, width*20/100)   // 20% of width, min 80px
	cornerH := max(30, height*10/100)  // 10% of height, min 30px

	zones := []scanZone{
		{"bottom-right", width - cornerW, height - cornerH, cornerW, cornerH, 1.5},
		{"bottom-left", 0, height - cornerH, cornerW, cornerH, 1.3},
		{"top-right", width - cornerW, 0, cornerW, cornerH, 1.2},
		{"top-left", 0, 0, cornerW, cornerH, 1.2},
		{"bottom-center", width/4, height - cornerH, width/2, cornerH, 1.0},
		{"top-center", width/4, 0, width/2, cornerH, 0.8},
	}

	var results []WatermarkRegion

	for _, zone := range zones {
		region := detectTextRegionInZone(frame, bounds, zone.x, zone.y, zone.w, zone.h)
		if region != nil {
			region.Score *= zone.priority
			log.Printf("[watermark] corner-scan: found candidate in %s → x=%d y=%d w=%d h=%d (score=%.2f)",
				zone.name, region.X, region.Y, region.W, region.H, region.Score)
			results = append(results, *region)
		}
	}

	return results
}

// detectTextRegionInZone looks for text-like patterns within a specific frame zone.
// Text watermarks have high local edge density — many sharp transitions in a small area.
func detectTextRegionInZone(frame image.Image, bounds image.Rectangle, zx, zy, zw, zh int) *WatermarkRegion {
	if zw < 8 || zh < 8 {
		return nil
	}

	// Compute grayscale values for the zone
	gray := make([][]uint8, zh)
	for py := 0; py < zh; py++ {
		gray[py] = make([]uint8, zw)
		for px := 0; px < zw; px++ {
			r, g, b, _ := frame.At(zx+px+bounds.Min.X, zy+py+bounds.Min.Y).RGBA()
			// Luminance formula
			lum := (r*299 + g*587 + b*114) / 1000
			gray[py][px] = uint8(lum >> 8)
		}
	}

	// Compute edge map using simple gradient magnitude (Sobel-like)
	blockSize := 4
	blocksW := zw / blockSize
	blocksH := zh / blockSize
	if blocksW < 2 || blocksH < 2 {
		return nil
	}

	// For each block, compute average gradient magnitude
	edgeDensity := make([][]float64, blocksH)
	totalEdge := 0.0
	for by := 0; by < blocksH; by++ {
		edgeDensity[by] = make([]float64, blocksW)
		for bx := 0; bx < blocksW; bx++ {
			var edgeSum float64
			count := 0
			for py := by*blockSize + 1; py < (by+1)*blockSize && py < zh-1; py++ {
				for px := bx*blockSize + 1; px < (bx+1)*blockSize && px < zw-1; px++ {
					// Simple gradient: |I(x+1) - I(x-1)| + |I(y+1) - I(y-1)|
					gx := int(gray[py][px+1]) - int(gray[py][px-1])
					gy := int(gray[py+1][px]) - int(gray[py-1][px])
					mag := math.Sqrt(float64(gx*gx + gy*gy))
					edgeSum += mag
					count++
				}
			}
			if count > 0 {
				edgeDensity[by][bx] = edgeSum / float64(count)
				totalEdge += edgeDensity[by][bx]
			}
		}
	}

	totalBlocks := blocksW * blocksH
	if totalBlocks == 0 {
		return nil
	}
	avgEdge := totalEdge / float64(totalBlocks)

	// Text blocks have medium-to-high edge density (not smooth, not extreme edges)
	// Look for clusters of blocks with edge density > 1.5x average
	textThreshold := avgEdge * 1.5
	if textThreshold < 5 {
		textThreshold = 5
	}

	// Find blocks above threshold
	textMask := make([][]bool, blocksH)
	textBlockCount := 0
	for by := 0; by < blocksH; by++ {
		textMask[by] = make([]bool, blocksW)
		for bx := 0; bx < blocksW; bx++ {
			if edgeDensity[by][bx] > textThreshold {
				textMask[by][bx] = true
				textBlockCount++
			}
		}
	}

	// Need at least a few text blocks (watermark text is small but not tiny)
	minBlocks := max(2, totalBlocks/20)  // at least 5% of zone blocks
	maxBlocks := totalBlocks * 60 / 100  // at most 60% (not the whole zone)

	if textBlockCount < minBlocks || textBlockCount > maxBlocks {
		return nil
	}

	// Find the bounding box of text blocks
	minBX, minBY := blocksW, blocksH
	maxBX, maxBY := 0, 0
	for by := 0; by < blocksH; by++ {
		for bx := 0; bx < blocksW; bx++ {
			if textMask[by][bx] {
				if bx < minBX { minBX = bx }
				if bx > maxBX { maxBX = bx }
				if by < minBY { minBY = by }
				if by > maxBY { maxBY = by }
			}
		}
	}

	// Convert block coordinates back to pixel coordinates in the zone
	rx := minBX * blockSize
	ry := minBY * blockSize
	rw := (maxBX-minBX+1)*blockSize
	rh := (maxBY-minBY+1)*blockSize

	// The region should be reasonably shaped (not too wide/tall relative to its other dimension)
	aspect := float64(rw) / float64(max(1, rh))
	if aspect > 15 || aspect < 0.2 {
		return nil
	}

	// Score based on text block density within the bounding box
	boxBlocks := (maxBX - minBX + 1) * (maxBY - minBY + 1)
	density := float64(textBlockCount) / float64(max(1, boxBlocks))

	score := density * 2.0
	if score < 0.5 {
		return nil
	}

	return &WatermarkRegion{
		X:     zx + rx,
		Y:     zy + ry,
		W:     rw,
		H:     rh,
		Score: score,
	}
}

// ──────────────────────────────────────────────────────────
//  Shared helpers
// ──────────────────────────────────────────────────────────

type wmComponent struct {
	minX, minY, maxX, maxY int
	size                   int
}

func wmFloodFill(mask, visited [][]bool, startX, startY, width, height int) wmComponent {
	comp := wmComponent{
		minX: startX, minY: startY,
		maxX: startX, maxY: startY,
	}

	type point struct{ x, y int }
	stack := []point{{startX, startY}}
	visited[startY][startX] = true

	dirs := [4][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}

	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		comp.size++
		if p.x < comp.minX { comp.minX = p.x }
		if p.x > comp.maxX { comp.maxX = p.x }
		if p.y < comp.minY { comp.minY = p.y }
		if p.y > comp.maxY { comp.maxY = p.y }

		for _, d := range dirs {
			nx, ny := p.x+d[0], p.y+d[1]
			if nx >= 0 && nx < width && ny >= 0 && ny < height &&
				!visited[ny][nx] && mask[ny][nx] {
				visited[ny][nx] = true
				stack = append(stack, point{nx, ny})
			}
		}
	}

	return comp
}

func wmScoreCandidate(c wmComponent, frameW, frameH int) float64 {
	cx := float64(c.minX+c.maxX) / 2.0
	cy := float64(c.minY+c.maxY) / 2.0

	edgeDistX := math.Min(cx, float64(frameW)-cx) / (float64(frameW) / 2.0)
	edgeDistY := math.Min(cy, float64(frameH)-cy) / (float64(frameH) / 2.0)
	edgeDist := math.Min(edgeDistX, edgeDistY)
	edgeScore := 1.0 - edgeDist

	area := float64(c.size)
	totalArea := float64(frameW * frameH)
	sizeRatio := area / totalArea

	var sizeScore float64
	switch {
	case sizeRatio >= 0.002 && sizeRatio <= 0.05:
		sizeScore = 1.0
	case sizeRatio < 0.002:
		sizeScore = sizeRatio / 0.002
	default:
		sizeScore = 0.05 / sizeRatio
	}

	boxArea := float64((c.maxX - c.minX + 1) * (c.maxY - c.minY + 1))
	compactness := area / boxArea

	return edgeScore*2.5 + sizeScore*1.5 + compactness*1.0
}

// deduplicateRegions removes overlapping regions, keeping the higher-scoring one.
func deduplicateRegions(regions []WatermarkRegion) []WatermarkRegion {
	if len(regions) <= 1 {
		return regions
	}

	// Sort by score descending
	sort.Slice(regions, func(i, j int) bool { return regions[i].Score > regions[j].Score })

	var kept []WatermarkRegion
	for _, r := range regions {
		overlaps := false
		for _, k := range kept {
			if regionsOverlap(r, k) {
				overlaps = true
				break
			}
		}
		if !overlaps {
			kept = append(kept, r)
		}
	}

	return kept
}

func regionsOverlap(a, b WatermarkRegion) bool {
	// Check if bounding boxes overlap significantly (>30% area overlap)
	overlapX := max(0, min(a.X+a.W, b.X+b.W)-max(a.X, b.X))
	overlapY := max(0, min(a.Y+a.H, b.Y+b.H)-max(a.Y, b.Y))
	overlapArea := overlapX * overlapY
	minArea := min(a.W*a.H, b.W*b.H)
	if minArea == 0 {
		return false
	}
	return float64(overlapArea)/float64(minArea) > 0.3
}
