package handler

import (
	"fmt"
	"github.com/benyaa/virtual-printer-process-engine/config"
	"github.com/benyaa/virtual-printer-process-engine/definitions"
	log "github.com/sirupsen/logrus"
)

func GetHandler(c config.HandlerConfig, idPrefix string) (definitions.Handler, error) {
	var handler definitions.Handler
	var err error
	switch c.Name {
	case "RunExecutable":
		handler, err = NewRunExecutableHandler(idPrefix, c.Config)
	case "MergePNGs":
		handler, err = NewMergePNGsHandler(idPrefix, c.Config)
	case "WriteFile":
		handler, err = NewWriteFileHandler(idPrefix, c.Config)
	case "ReadFile":
		handler, err = NewReadFileHandler(idPrefix, c.Config)
	case "UploadHTTP":
		handler, err = NewSendHTTPHandler(idPrefix, c.Config)
	default:
		return nil, fmt.Errorf("unknown handler name")
	}

	if err != nil {
		log.WithError(err).Error("failed to create handler")
		return nil, err
	}

	return handler, nil
}
