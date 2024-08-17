package definitions

import (
	"github.com/benyaa/virtual-printer-process-engine/utils"
	"github.com/mitchellh/mapstructure"
	"io"
)

type EngineFlowObject struct {
	Pages    int                    `json:"pages"`
	Metadata map[string]interface{} `json:"metadata"`
}

func (e *EngineFlowObject) EvaluateExpression(input string) (string, error) {
	return utils.EvaluateExpression(input, e.Metadata)
}

type BaseHandler struct {
	ID string
}

func (b *BaseHandler) GetID() string {
	return b.ID
}

func (b *BaseHandler) DecodeMap(input interface{}, output interface{}) error {
	return mapstructure.Decode(input, output)
}

type Handler interface {
	GetID() string
	Name() string
	Handle(info *EngineFlowObject, fileHandler EngineFileHandler) (*EngineFlowObject, error)
}

type EngineFileHandler interface {
	Read() (io.Reader, error)
	Write() (io.Writer, error)
	Close()
}
