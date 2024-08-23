package handler

import (
	"github.com/benyaa/virtual-printer-process-engine/definitions"
	log "github.com/sirupsen/logrus"
	"image/jpeg"
	"image/png"
	"os"
)

type ConvertPNGToJPEGHandler struct {
	definitions.BaseHandler
	config *convertPNGToJPEGConfig
}

type convertPNGToJPEGConfig struct {
	InputFile      string `mapstructure:"input_file"`
	OutputFile     string `mapstructure:"output_file"`
	RemoveOriginal bool   `mapstructure:"remove_original,omitempty"`
}

func NewConvertPNGToJPEGHandler(idPrefix string, c map[string]interface{}) (*ConvertPNGToJPEGHandler, error) {
	h := &ConvertPNGToJPEGHandler{
		BaseHandler: definitions.BaseHandler{
			ID: idPrefix + "_convert_png_to_jpeg",
		},
	}
	err := h.setConfig(c)
	if err != nil {
		return nil, err
	}

	return h, nil
}

func (h *ConvertPNGToJPEGHandler) setConfig(config map[string]interface{}) error {
	h.config = &convertPNGToJPEGConfig{}
	return h.DecodeMap(config, h.config)
}

func (h *ConvertPNGToJPEGHandler) Name() string {
	return "ConvertPNGToJPEG"
}

func (h *ConvertPNGToJPEGHandler) Handle(info *definitions.EngineFlowObject, fileHandler definitions.EngineFileHandler) (*definitions.EngineFlowObject, error) {
	log.Debugf("evaluating input file %s", h.config.InputFile)
	input, err := info.EvaluateExpression(h.config.InputFile)
	if err != nil {
		log.WithError(err).Errorf("failed to evaluate input file %s", h.config.InputFile)
		return nil, err
	}
	log.Debugf("evaluated input file %s", input)

	log.Debugf("evaluating output file %s", h.config.OutputFile)
	output, err := info.EvaluateExpression(h.config.OutputFile)
	if err != nil {
		log.WithError(err).Errorf("failed to evaluate output file %s", h.config.OutputFile)
		return nil, err
	}
	log.Debugf("evaluated output file %s", output)

	pngFile, err := os.Open(input)
	if err != nil {
		log.WithError(err).Errorf("failed to open input file %s", h.config.InputFile)
		return nil, err
	}

	pngImage, err := png.Decode(pngFile)
	if err != nil {
		log.WithError(err).Errorf("failed to decode png file %s", h.config.InputFile)
		return nil, err
	}

	jpegFile, err := os.Create(output)
	if err != nil {
		log.WithError(err).Errorf("failed to create output file %s", h.config.OutputFile)
		return nil, err
	}

	err = jpeg.Encode(jpegFile, pngImage, &jpeg.Options{
		Quality: 100,
	})
	if err != nil {
		log.WithError(err).Errorf("failed to encode jpeg file %s", h.config.OutputFile)
		return nil, err
	}

	err = pngFile.Close()

	if h.config.RemoveOriginal {
		if err != nil {
			log.WithError(err).Errorf("failed to close input file %s", h.config.InputFile)
			return nil, err
		}
		err = os.Remove(h.config.InputFile)
		if err != nil {
			log.WithError(err).Errorf("failed to remove original file %s", h.config.InputFile)
			return nil, err
		}
	}

	info.Metadata["ConvertPNGToJPEG.OutputFile"] = output

	return info, nil
}
