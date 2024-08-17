package handler

import (
	"github.com/benyaa/virtual-printer-process-engine/definitions"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
)

/**
 * WriteFileHandler is a handler that writes the input file to a specified output file
 * Metadata:
			 * Write:
					* `WriteFile.OutputPath` - the path to the output file
*/
type WriteFileHandler struct {
	definitions.BaseHandler
	config *writeFileHandlerConfig
}

func NewWriteFileHandler(idPrefix string, c map[string]interface{}) (*WriteFileHandler, error) {
	h := &WriteFileHandler{
		BaseHandler: definitions.BaseHandler{
			ID: idPrefix + "_write_file",
		},
	}
	err := h.setConfig(c)
	if err != nil {
		return nil, err
	}

	return h, nil
}

type writeFileHandlerConfig struct {
	Output string `mapstructure:"output"`
}

func (h *WriteFileHandler) Name() string {
	return "WriteFile"
}

func (h *WriteFileHandler) setConfig(config map[string]interface{}) error {
	h.config = &writeFileHandlerConfig{}
	return h.DecodeMap(config, h.config)
}

func (h *WriteFileHandler) Handle(info *definitions.EngineFlowObject, fileHandler definitions.EngineFileHandler) (*definitions.EngineFlowObject, error) {
	log.Trace("WriteFileHandler.Handle")
	log.Debugf("writing file to %s", h.config.Output)
	reader, err := fileHandler.Read()
	if err != nil {
		return nil, err

	}
	log.Debugf("evaluating expression %s", h.config.Output)
	outputPath, err := info.EvaluateExpression(h.config.Output)
	if err != nil {
		return nil, err
	}
	log.Debugf("evaluated expression to %s", outputPath)

	err = os.MkdirAll(filepath.Dir(outputPath), os.ModePerm)
	if err != nil {
		return nil, err
	}

	log.Debugf("creating file %s", outputPath)

	writer, err := os.Create(outputPath)
	if err != nil {
		return nil, err
	}
	defer writer.Close()

	log.Debugf("copying file to %s", outputPath)
	_, err = io.Copy(writer, reader)
	if err != nil {
		return nil, err
	}

	log.Debugf("setting metadata WriteFile.OutputPath to %s", outputPath)

	info.Metadata["WriteFile.OutputPath"] = outputPath

	return info, nil
}
