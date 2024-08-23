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

func (h *UploadHTTPHandler) generateStreamingRequest(url string, info *definitions.EngineFlowObject, reader io.Reader) (*http.Request, error) {
	pr, pw := io.Pipe()
	req, err := http.NewRequest("POST", url, pr)
	if err != nil {
		log.WithError(err).Errorf("failed to create HTTP request")
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	switch h.config.Type {
	case sendFileMultipart:
		log.Debugf("sending file as multipart with streaming")
		writer := multipart.NewWriter(pw)
		contentType := writer.FormDataContentType()
		req.Header.Set("Content-Type", contentType)
		log.Debugf("generating multipart with content type %s", contentType)
		go func() {
			err := h.generateMultipart(info, writer, reader)
			if err != nil {
				pw.CloseWithError(err)
				return
			}

			log.Debugf("setting content type as %s", contentType)
			pw.Close()
		}()
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
		go func() {
			_, err = pw.Write([]byte(formattedContent))
			if err != nil {
				pw.CloseWithError(err)
				log.WithError(err).Errorf("failed to write formatted content")
			}
			pw.Close()
		}()
	}

	return req, nil
}
