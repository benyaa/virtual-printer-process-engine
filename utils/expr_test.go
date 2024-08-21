package utils

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEvaluateExpression_SingleExpression(t *testing.T) {
	data := map[string]interface{}{
		"test": "result",
	}
	result, err := EvaluateExpression("${test}", data)
	assert.NoError(t, err)
	assert.Equal(t, "result", result)
}

func TestEvaluateExpression_MultipleExpressions(t *testing.T) {
	data := map[string]interface{}{
		"test":  "result",
		"test2": "result2",
	}
	result, err := EvaluateExpression("${test}/${test2}/do", data)
	assert.NoError(t, err)
	assert.Equal(t, "result/result2/do", result)
}

func TestEvaluateExpression_WithUUID(t *testing.T) {
	data := map[string]interface{}{}
	result, err := EvaluateExpression("${uuid()}", data)
	assert.NoError(t, err)
	_, err = uuid.Parse(result)
	assert.NoError(t, err)
}

func TestEvaluateExpression_WithBackslashExpression(t *testing.T) {
	data := map[string]interface{}{
		"test": "result",
	}
	result, err := EvaluateExpression(`\${test}`, data)
	assert.NoError(t, err)
	assert.Equal(t, "${test}", result)
}

func TestEvaluateExpression_WithDoubleBackslashExpression(t *testing.T) {
	data := map[string]interface{}{
		"test": "result",
	}
	result, err := EvaluateExpression(`\\${test}`, data)
	assert.NoError(t, err)
	assert.Equal(t, `\result`, result)
}

func TestEvaluateExpression_WithBackslashInnerExpression(t *testing.T) {
	data := map[string]interface{}{
		"test": "result",
	}
	result, err := EvaluateExpression(`\${test${test}}`, data)
	assert.NoError(t, err)
	assert.Equal(t, "${testresult}", result)
}
