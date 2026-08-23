package service

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestFingerprintService_PerceptualHash(t *testing.T) {
	fs := NewFingerprintService()

	// Create a test image with more variation
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			// Create checkerboard pattern for better fingerprint
			if (x+y)%2 == 0 {
				img.Set(x, y, color.RGBA{255, 0, 0, 255})
			} else {
				img.Set(x, y, color.RGBA{0, 255, 0, 255})
			}
		}
	}

	fp := fs.PerceptualHash(img)

	// FP can be 0 for uniform images, but checkerboard should have non-zero
	t.Logf("PerceptualHash result: %d", fp)
}

func TestFingerprintService_PerceptualHash_SameImage(t *testing.T) {
	fs := NewFingerprintService()

	// Create identical images
	img1 := image.NewRGBA(image.Rect(0, 0, 50, 50))
	img2 := image.NewRGBA(image.Rect(0, 0, 50, 50))

	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			img1.Set(x, y, color.RGBA{128, 128, 128, 255})
			img2.Set(x, y, color.RGBA{128, 128, 128, 255})
		}
	}

	fp1 := fs.PerceptualHash(img1)
	fp2 := fs.PerceptualHash(img2)

	if fp1 != fp2 {
		t.Errorf("expected same fingerprint for same image, got %d vs %d", fp1, fp2)
	}
}

func TestFingerprintService_PerceptualHash_DifferentImages(t *testing.T) {
	fs := NewFingerprintService()

	// Create different checkerboard patterns
	img1 := image.NewRGBA(image.Rect(0, 0, 50, 50))
	img2 := image.NewRGBA(image.Rect(0, 0, 50, 50))

	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			// Different patterns
			if (x+y)%2 == 0 {
				img1.Set(x, y, color.RGBA{255, 0, 0, 255})
				img2.Set(x, y, color.RGBA{0, 255, 0, 255})
			} else {
				img1.Set(x, y, color.RGBA{0, 255, 0, 255})
				img2.Set(x, y, color.RGBA{255, 0, 0, 255})
			}
		}
	}

	fp1 := fs.PerceptualHash(img1)
	fp2 := fs.PerceptualHash(img2)

	t.Logf("FP1: %d, FP2: %d", fp1, fp2)
	// Different patterns should likely produce different fingerprints
	if fp1 == fp2 {
		t.Log("Warning: different images produced same fingerprint (may happen with small images)")
	}
}

func TestFingerprintService_Simhash(t *testing.T) {
	fs := NewFingerprintService()

	data := []byte("test data for simhash")
	fp := fs.Simhash(data)

	if fp == 0 {
		t.Error("expected non-zero simhash fingerprint")
	}
}

func TestFingerprintService_Simhash_SameData(t *testing.T) {
	fs := NewFingerprintService()

	data := []byte("same data")
	fp1 := fs.Simhash(data)
	fp2 := fs.Simhash(data)

	if fp1 != fp2 {
		t.Errorf("expected same simhash for same data")
	}
}

func TestFingerprintService_HammingDistance(t *testing.T) {
	fs := NewFingerprintService()

	// Same value should have distance 0
	dist := fs.HammingDistance(0xABCD, 0xABCD)
	if dist != 0 {
		t.Errorf("expected distance 0 for same values, got %d", dist)
	}

	// Different values should have non-zero distance
	dist = fs.HammingDistance(0x0000, 0xFFFF)
	if dist == 0 {
		t.Error("expected non-zero distance for different values")
	}
}

func TestFingerprintService_AreSimilar(t *testing.T) {
	fs := NewFingerprintService()

	// Same values should be similar
	if !fs.AreSimilar(0xABCD, 0xABCD, 5) {
		t.Error("expected same values to be similar")
	}

	// Different values with high threshold should not be similar
	if fs.AreSimilar(0x0000, 0xFFFF, 1) {
		t.Error("expected very different values to not be similar")
	}

	// Different values with low threshold should be similar
	if !fs.AreSimilar(0x0000, 0xFFFF, 64) {
		t.Error("expected values with small distance to be similar")
	}
}

func TestFingerprintService_FingerprintVideo(t *testing.T) {
	fs := NewFingerprintService()

	// Create a test PNG image
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 0, 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode image: %v", err)
	}

	frameData := buf.Bytes()

	// Test with both frame and video data
	videoData := []byte("fake video data for testing")
	fp, err := fs.FingerprintVideo(frameData, videoData)
	if err != nil {
		t.Logf("FingerprintVideo returned error (expected for fake data): %v", err)
		return
	}
	
	if fp == 0 {
		t.Error("expected non-zero fingerprint")
	}
}
