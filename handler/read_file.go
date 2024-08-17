package handler

import (
	"github.com/benyaa/virtual-printer-process-engine/definitions"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
)

/**
 * ReadFileHandler is a handler that reads the input file to a specified output file
 * Metadata:
			 * Write:
					* `ReadFile.Source` - the path to the input file
*/
type ReadFileHandler struct {
	definitions.BaseHandler
	config *ReadFileHandlerConfig
}

func NewReadFileHandler(idPrefix string, c map[string]interface{}) (*ReadFileHandler, error) {
	h := &ReadFileHandler{
		BaseHandler: definitions.BaseHandler{
			ID: idPrefix + "_read_file",
		},
	}
	err := h.setConfig(c)
	if err != nil {
		return nil, err
	}
	return h, nil
}

type ReadFileHandlerConfig struct {
	Input        string `mapstructure:"input"`
	RemoveSource bool   `mapstructure:"remove_source"`
}

func (h *ReadFileHandler) Name() string {
	return "ReadFile"
}

func (h *ReadFileHandler) setConfig(config map[string]interface{}) error {
	h.config = &ReadFileHandlerConfig{}
	return h.DecodeMap(config, h.config)
}

func (h *ReadFileHandler) Handle(info *definitions.EngineFlowObject, fileHandler definitions.EngineFileHandler) (*definitions.EngineFlowObject, error) {
	log.Trace("handling ReadFile")
	writer, err := fileHandler.Write()
	if err != nil {
		return nil, err
	}

	log.Debugf("evaluating expression %s", h.config.Input)
	inputPath, err := info.EvaluateExpression(h.config.Input)
	if err != nil {
		return nil, err
	}
	log.Debugf("input path: %s", inputPath)

	reader, err := os.Open(inputPath)
	if err != nil {
		return nil, err
	}

	log.Debugf("copying %s to output", inputPath)
	_, err = io.Copy(writer, reader)
	if err != nil {
		return nil, err
	}

	reader.Close()
	if h.config.RemoveSource {
		log.Debugf("removing source file %s", inputPath)
		err = os.Remove(inputPath)
		if err != nil {
			return nil, err
		}
	}
	log.Debugf("setting metadata ReadFile.Source to %s", inputPath)

	info.Metadata["ReadFile.Source"] = inputPath

	return info, nil
}
