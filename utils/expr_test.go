package utils

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEvaluateExpressionSingleExpression(t *testing.T) {
	data := map[string]interface{}{
		"test": "result",
	}
	result, err := EvaluateExpression("${test}", data)
	assert.NoError(t, err)
	assert.Equal(t, "result", result)
}

func TestEvaluateExpressionMultipleExpressions(t *testing.T) {
	data := map[string]interface{}{
		"test":  "result",
		"test2": "result2",
	}
	result, err := EvaluateExpression("${test}/${test2}/do", data)
	assert.NoError(t, err)
	assert.Equal(t, "result/result2/do", result)
}

func TestEvaluateExpressionWithUUID(t *testing.T) {
	data := map[string]interface{}{}
	result, err := EvaluateExpression("${uuid()}", data)
	assert.NoError(t, err)
	_, err = uuid.Parse(result)
	assert.NoError(t, err)
}

func TestEvaluateExpressionWithBackslashExpression(t *testing.T) {
	data := map[string]interface{}{
		"test": "result",
	}
	result, err := EvaluateExpression(`\${test}`, data)
	assert.NoError(t, err)
	assert.Equal(t, "${test}", result)
}

func TestEvaluateExpressionWithDoubleBackslashExpression(t *testing.T) {
	data := map[string]interface{}{
		"test": "result",
	}
	result, err := EvaluateExpression(`\\${test}`, data)
	assert.NoError(t, err)
	assert.Equal(t, `\result`, result)
}

func TestEvaluateExpressionWithBackslashInnerExpression(t *testing.T) {
	data := map[string]interface{}{
		"test": "result",
	}
	result, err := EvaluateExpression(`\${test${test}}`, data)
	assert.NoError(t, err)
	assert.Equal(t, "${testresult}", result)
}
