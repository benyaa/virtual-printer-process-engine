package utils

import (
	"fmt"
	"github.com/expr-lang/expr"
	"github.com/google/uuid"
	"os"
	"strings"
)

var EvaluateExpression = evaluateExpression

var uuidFunc = expr.Function("uuid", func(params ...any) (any, error) {
	return uuid.NewString(), nil
})

var envVarFunc = expr.Function("getEnv", func(params ...any) (any, error) {
	if len(params) != 1 {
		return nil, fmt.Errorf("env function expects exactly 1 argument")
	}

	key, ok := params[0].(string)
	if !ok {
		return nil, fmt.Errorf("env function expects a string argument")
	}

	return os.Getenv(key), nil
})

func evaluateExpression(input string, data map[string]interface{}) (string, error) {
	var result strings.Builder
	var expression strings.Builder
	inExpression := false
	escapeNext := false

	for i := 0; i < len(input); i++ {
		char := input[i]

		if escapeNext {
			if inExpression {
				expression.WriteByte(char)
			} else {
				result.WriteByte(char)
			}
			escapeNext = false
			continue
		}

		if char == '\\' {
			escapeNext = true
			continue
		}

		if char == '$' && i+1 < len(input) && input[i+1] == '{' {
			if inExpression {
				return "", fmt.Errorf("nested expressions are not supported, use the `+` operator instead")
			}
			inExpression = true
			i++ // skip '{'
			continue
		}

		if char == '}' && inExpression {
			exprStr := expression.String()
			program, err := expr.Compile(exprStr, expr.Env(data), uuidFunc, envVarFunc)
			if err != nil {
				return "", fmt.Errorf("failed to compile expression: %v", err)
			}

			exprResult, err := expr.Run(program, data)
			if err != nil {
				return "", fmt.Errorf("failed to run expression: %v", err)
			}

			result.WriteString(fmt.Sprintf("%v", exprResult))
			expression.Reset()
			inExpression = false
			continue
		}

		if inExpression {
			expression.WriteByte(char)
		} else {
			result.WriteByte(char)
		}
	}

	// If we end with an unfinished expression, return an error
	if inExpression {
		return "", fmt.Errorf("unfinished expression found")
	}

	return result.String(), nil
}
