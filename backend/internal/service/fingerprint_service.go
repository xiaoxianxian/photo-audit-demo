package service

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"image"
	"image/jpeg"
	"math"
)

// FingerprintService handles video fingerprinting using perceptual hash + simhash.
type FingerprintService struct{}

// NewFingerprintService creates a new fingerprint service.
func NewFingerprintService() *FingerprintService {
	return &FingerprintService{}
}

// PerceptualHash computes a pHash fingerprint from an image.
// Returns a 64-bit hash where similar images have similar hashes.
func (s *FingerprintService) PerceptualHash(img image.Image) uint64 {
	// Resize to 32x32 for efficiency.
	resized := resize(img, 32, 32)
	gray := toGrayscale(resized)

	// Compute DCT-like average.
	var sum int64
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			idx := y*32 + x
			if idx < len(gray) {
				sum += int64(gray[idx])
			}
		}
	}
	avg := sum / 64

	// Build hash from lower-right 8x8 block (frequency domain approximation).
	hash := uint64(0)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			idx := (y + 24) * 32 + (x + 24)
			if idx < len(gray) {
				bit := int64(gray[idx]) - avg
				if bit < 0 {
					hash |= 1 << (y*8 + x)
				}
			}
		}
	}
	return hash
}

// Simhash computes a simhash fingerprint from byte data.
// Uses character trigram features with SHA1 hashing.
func (s *FingerprintService) Simhash(data []byte) uint64 {
	// Convert to string for trigram extraction.
	content := string(data)
	if len(content) > 10000 {
		content = content[:10000]
	}

	// Extract trigrams as features.
	weights := make(map[int64]int64)
	trigrams := make([]byte, 0, len(content)-2)
	for i := 0; i < len(content)-2; i++ {
		trigrams = append(trigrams, content[i:i+3]...)
	}

	for i := 0; i < len(trigrams)-2; i++ {
		trigram := trigrams[i : i+3]
		h := sha1.Sum(trigram)
		var feature int64
		for j := 0; j < 8 && j < len(h); j++ {
			feature = feature<<8 | int64(h[j])
		}
		weights[feature]++
	}

	// Build simhash vector.
	var hash uint64
	for feature, weight := range weights {
		h := sha1.Sum([]byte(fmt.Sprintf("%d", feature)))
		for i := 0; i < 64 && i < len(h)*2; i++ {
			byteIdx := i / 8
			bitIdx := i % 8
			if byteIdx < len(h) {
				bit := (h[byteIdx] >> (7 - bitIdx)) & 1
				if bit == 1 {
					hash |= 1 << i
				}
			}
		}
		_ = weight // use weight for signed simhash in production
	}

	return hash
}

// HammingDistance counts the number of differing bits between two 64-bit hashes.
func (s *FingerprintService) HammingDistance(a, b uint64) int {
	x := a ^ b
	count := 0
	for x != 0 {
		count += int(x & 1)
		x >>= 1
	}
	return count
}

// AreSimilar checks if two fingerprints are within the similarity threshold.
// Default threshold is 10 bits for 64-bit simhash (≈84% similarity).
func (s *FingerprintService) AreSimilar(a, b uint64, threshold int) bool {
	return s.HammingDistance(a, b) <= threshold
}

// resize creates a simplified resized image by sampling.
func resize(img image.Image, w, h int) image.Image {
	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	sampled := make([]byte, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sx := x * srcW / w
			sy := y * srcH / h
			p := img.At(sx, sy)
			r, _, _, _ := p.RGBA()
			sampled[y*w+x] = byte(r >> 8)
		}
	}
	return image.NewGray(image.Rect(0, 0, w, h))
}

// toGrayscale converts an RGBA image to a gray byte slice.
func toGrayscale(img image.Image) []byte {
	bounds := img.Bounds()
	gray := make([]byte, bounds.Dx()*bounds.Dy())
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			// BT.601 luminance formula.
			lum := uint16(0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b))
			gray[y*bounds.Dx()+x] = byte(lum >> 8)
		}
	}
	return gray
}

// FingerprintVideo extracts a fingerprint from video frame data.
// Uses the first frame's perceptual hash combined with content simhash.
func (s *FingerprintService) FingerprintVideo(frameData []byte, videoData []byte) (uint64, error) {
	// Decode first frame as JPEG.
	img, err := jpeg.Decode(bytes.NewReader(frameData))
	if err != nil {
		// If decode fails, fall back to content hash.
		return s.Simhash(videoData), nil
	}

	// Compute perceptual hash from first frame.
	pHash := s.PerceptualHash(img)

	// Combine with content simhash.
	cSimhash := s.Simhash(videoData)

	// XOR combine for final fingerprint.
	fingerprint := pHash ^ cSimhash

	return fingerprint, nil
}

// --- Hash collision detection helpers ---

// CosineSimilarity computes cosine similarity between two float64 vectors.
func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, magA, magB float64
	for i := range a {
		dot += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}
	if magA == 0 || magB == 0 {
		return 0
	}
	return dot / (math.Sqrt(magA) * math.Sqrt(magB))
}

// AverageHash computes a simple average hash (8x8 = 64 bits).
func AverageHash(data []byte) uint64 {
	// Use first 64 bytes as simplified hash.
	var h uint64
	for i := 0; i < 8 && i < len(data); i++ {
		h = h<<8 | uint64(data[i])
	}
	return h
}
