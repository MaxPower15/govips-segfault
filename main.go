package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"

	"github.com/MaxPower15/govips-segfault/parmap"
	"github.com/MaxPower15/govips-segfault/seq"
	"github.com/davidbyttow/govips/v2/vips"
)

func main() {
	items := []int{}

	totalOps := 91 * 800
	for i := 0; i < totalOps; i++ {
		items = append(items, i)
	}

	vips.LoggingSettings(nil, vips.LogLevelError)
	vips.Startup(nil)

	_, err := parmap.IntsToStrings(context.Background(), items, 100, func(item, index int) (string, error) {
		frameIndex := item%91 + 1

		set1Path := fmt.Sprintf("fixtures/frameset1/%05d.jpg", frameIndex)
		set2Path := fmt.Sprintf("fixtures/frameset2/%05d.jpg", frameIndex)

		outputIndex := seq.NextAsInt()
		outputPath := fmt.Sprintf("output/%08d.jpg", outputIndex)

		fmt.Printf("%05d.jpg -> output/%08d.jpg\n", frameIndex, outputIndex)

		canvas, err := mkBlankImg(2560, 1440, 0)
		if err != nil {
			return "", fmt.Errorf("making canvas: %w", err)
		}
		defer canvas.Close()

		file1, err := newThumbnailFromFile(set1Path, 1280, 720)
		if err != nil {
			return "", fmt.Errorf("loading file1: %w", err)
		}
		defer file1.Close()

		file2, err := newThumbnailFromFile(set2Path, 1280, 720)
		if err != nil {
			return "", fmt.Errorf("loading file2: %w", err)
		}
		defer file2.Close()

		if err := insert(canvas, file1, 0, 360); err != nil {
			return "", fmt.Errorf("inserting file1: %w", err)
		}

		if err := insert(canvas, file2, 1280, 360); err != nil {
			return "", fmt.Errorf("inserting file2: %w", err)
		}

		if err := saveAsJpeg(canvas, outputPath); err != nil {
			return "", fmt.Errorf("saving as jpeg: %w", err)
		}

		return "", nil
	})

	if err != nil {
		log.Fatal(err)
	}
}

func mkBlankImg(width, height, rgb int) (*vips.ImageRef, error) {
	img, err := vips.Black(1, 1)
	if err != nil {
		return nil, err
	}

	err = img.Linear1(1, float64(rgb))
	if err != nil {
		return nil, err
	}

	err = img.Embed(0, 0, width, height, vips.ExtendRepeat)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func saveAsJpeg(img *vips.ImageRef, path string) error {
	bytes, _, err := img.ExportJpeg(&vips.JpegExportParams{Quality: 92})
	if err != nil {
		return err
	}

	err = os.WriteFile(path, bytes, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func newThumbnailFromFile(path string, width float64, height float64) (*vips.ImageRef, error) {
	img, err := vips.NewImageFromFile(path)
	if err != nil {
		return nil, err
	}

	err = img.Thumbnail(int(width), int(height), vips.InterestingCentre)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func insert(canvas, overlay *vips.ImageRef, left, top float64) error {
	if overlay == nil {
		fmt.Println("insert: overlay is nil!")
	}
	err := canvas.Insert(
		overlay,
		int(math.Round(left)),
		int(math.Round(top)),
		false,
		nil,
	)
	return err
}
