package handler

import (
	"fmt"
	"github.com/benyaa/virtual-printer-process-engine/definitions"
	"github.com/benyaa/virtual-printer-process-engine/utils"
	log "github.com/sirupsen/logrus"
)

/*
*
  - RunExecutableHandler is a handler that runs an executable with the given arguments
*/
type RunExecutableHandler struct {
	definitions.BaseHandler
	config *runExecConfig
}

func NewRunExecutableHandler(idPrefix string, c map[string]interface{}) (*RunExecutableHandler, error) {
	handler := &RunExecutableHandler{
		BaseHandler: definitions.BaseHandler{
			ID: fmt.Sprintf("%s_run_executable", idPrefix),
		},
	}
	err := handler.setConfig(c)
	if err != nil {
		return nil, err
	}
	return handler, nil
}

type runExecConfig struct {
	Executable string   `mapstructure:"executable"`
	Args       []string `mapstructure:"args"`
}

func (h *RunExecutableHandler) Name() string {
	return "RunExecutable"
}

func (h *RunExecutableHandler) setConfig(config map[string]interface{}) error {
	h.config = &runExecConfig{}
	return h.DecodeMap(config, h.config)
}

func (h *RunExecutableHandler) Handle(info *definitions.EngineFlowObject, fileHandler definitions.EngineFileHandler) (*definitions.EngineFlowObject, error) {
	var err error
	// convert templated args to actual args
	parsedArgs := make([]string, len(h.config.Args))
	for i, arg := range h.config.Args {
		parsedArgs[i], err = info.EvaluateExpression(arg)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate expression for arg %s: %w", arg, err)
		}
	}

	log.Debugf("Converting file using executable: %s args: %v", h.config.Executable, parsedArgs)

	output, err := utils.ExecuteCommand(h.config.Executable, parsedArgs...)
	if err != nil {
		log.WithError(err).Errorf("failed to run executable %s: %s", h.config.Executable, output)
		return nil, fmt.Errorf("failed to run executable %s: %w. Output: %s", h.config.Executable, err, output)
	}
	return info, nil
}
