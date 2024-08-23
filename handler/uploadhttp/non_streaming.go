package uploadhttp

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/benyaa/virtual-printer-process-engine/definitions"
	log "github.com/sirupsen/logrus"
	"io"
	"mime/multipart"
	"net/http"
)

func (h *UploadHTTPHandler) generateMemoryLoaderRequest(url string, info *definitions.EngineFlowObject, reader io.Reader) (*http.Request, error) {
	var requestBody bytes.Buffer
	req, err := http.NewRequest("POST", url, &requestBody)
	if err != nil {
		log.WithError(err).Errorf("failed to create HTTP request")
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	switch h.config.Type {
	case sendFileMultipart:
		log.Debugf("sending file as multipart with memory loader")
		writer := multipart.NewWriter(&requestBody)
		contentType := writer.FormDataContentType()
		req.Header.Set("Content-Type", contentType)
		log.Debugf("generating multipart with content type %s", contentType)
		err := h.generateMultipart(info, writer, reader)
		if err != nil {
			return nil, err
		}
	case sendFileBase64:
		log.Debugf("Sending file as base64")
		var base64Content bytes.Buffer
		base64Writer := base64.NewEncoder(base64.StdEncoding, &base64Content)
		defer base64Writer.Close()
		log.Debugf("copying file to base64")
		_, err = io.Copy(base64Writer, reader)
		if err != nil {
			log.WithError(err).Errorf("failed to copy file to base64")
			return nil, fmt.Errorf("failed to copy file to base64: %w", err)
		}
		log.Debugf("closing base64 writer")
		formattedContent, err := h.formatBase64Content(base64Content.String(), info)
		if err != nil {
			log.WithError(err).Errorf("failed to format base64 content")
			return nil, fmt.Errorf("failed to format base64 content: %w", err)
		}
		_, err = requestBody.Write([]byte(formattedContent))
		if err != nil {
			log.WithError(err).Errorf("failed to write formatted content")
			return nil, fmt.Errorf("failed to write formatted content: %w", err)
		}
	}

	return req, nil
}
