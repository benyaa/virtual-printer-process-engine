package handler

import (
	"fmt"
	"github.com/benyaa/virtual-printer-process-engine/definitions"
	log "github.com/sirupsen/logrus"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

type MergePNGsHandler struct {
	definitions.BaseHandler
	config *mergePNGsConfig
}

func NewMergePNGsHandler(idPrefix string, c map[string]interface{}) (*MergePNGsHandler, error) {
	h := &MergePNGsHandler{
		BaseHandler: definitions.BaseHandler{
			ID: fmt.Sprintf("%s_merge_pngs", idPrefix),
		},
	}
	err := h.setConfig(c)
	if err != nil {
		return nil, err
	}

	return h, nil
}

type mergePNGsConfig struct {
	InputFile      string `mapstructure:"input_file"`
	OutputFile     string `mapstructure:"output_file"`
	RemoveOldFiles bool   `mapstructure:"remove_old_files,omitempty"`
}

func (h *MergePNGsHandler) setConfig(config map[string]interface{}) error {
	h.config = &mergePNGsConfig{}
	return h.DecodeMap(config, h.config)
}

func (h *MergePNGsHandler) Name() string {
	return "MergePNGs"
}

func (h *MergePNGsHandler) Handle(info *definitions.EngineFlowObject, fileHandler definitions.EngineFileHandler) (*definitions.EngineFlowObject, error) {
	numPages := info.Pages

	if numPages < 1 {
		return nil, fmt.Errorf("no pages to merge")
	}

	basePath, err := info.EvaluateExpression(h.config.InputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate input file: %w", err)
	}

	dir := filepath.Dir(basePath)
	ext := filepath.Ext(basePath)
	base := strings.TrimSuffix(filepath.Base(basePath), ext)
	log.Debugf("Got dir as %s, ext as %s, base as %s", dir, ext, base)

	// Open the first image to get the dimensions
	firstImgFile, err := os.Open(filepath.Join(dir, fmt.Sprintf("%s1%s", base, ext)))
	if err != nil {
		return nil, fmt.Errorf("failed to open first image: %w", err)
	}
	defer firstImgFile.Close()

	firstImg, err := png.Decode(firstImgFile)
	if err != nil {
		return nil, fmt.Errorf("failed to decode first image: %w", err)
	}

	// Calculate the size of the final image
	width := firstImg.Bounds().Dx()
	height := firstImg.Bounds().Dy() * numPages

	// Create a new blank image with the combined height
	finalImage := image.NewRGBA(image.Rect(0, 0, width, height))

	// Draw each image onto the final image
	for i := 1; i <= numPages; i++ {
		imgFilePath := filepath.Join(dir, fmt.Sprintf("%s%d%s", base, i, ext))
		imgFile, err := os.Open(imgFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open image %s: %w", imgFilePath, err)
		}
		defer imgFile.Close()

		img, err := png.Decode(imgFile)
		if err != nil {
			return nil, fmt.Errorf("failed to decode image %s: %w", imgFilePath, err)
		}

		// Calculate the position where the image should be drawn
		offset := image.Pt(0, (i-1)*firstImg.Bounds().Dy())
		draw.Draw(finalImage, img.Bounds().Add(offset), img, image.Point{0, 0}, draw.Src)
	}

	// Save the final image to the output file
	outFile, err := os.Create(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	if err := png.Encode(outFile, finalImage); err != nil {
		return nil, fmt.Errorf("failed to encode final image: %w", err)
	}

	// cleanup the temporary files
	for i := 1; i <= numPages; i++ {
		imgFilePath := filepath.Join(dir, fmt.Sprintf("%s%d%s", base, i, ext))
		err = os.Remove(imgFilePath)
		if err != nil {
			log.WithError(err).Warnf("failed to remove temporary file %s", imgFilePath)
		}
	}

	info.Metadata["MergePNGs.OutputFile"] = basePath

	return info, nil
}
