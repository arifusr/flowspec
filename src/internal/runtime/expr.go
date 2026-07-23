package runtime

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/testing-cli/apitest/internal/ast"
)

// ExprContext holds context for evaluating expressions within a transform map block.
type ExprContext struct {
	Element        map[string]interface{} // current source element (item.*)
	ComputedFields map[string]interface{} // previously computed fields in this map iteration
	Vars           *Variables             // variable scope for {{var}} resolution
	FieldName      string                 // target field name (for error messages)
	Warnings       []string               // collected warnings
}

// EvalExpr evaluates an expression AST node and returns the result as float64.
func EvalExpr(expr ast.Expr, ctx *ExprContext) (float64, error) {
	switch e := expr.(type) {
	case *ast.NumberLitExpr:
		f, err := strconv.ParseFloat(e.Value, 64)
		if err != nil {
			ctx.addWarning("cannot parse number literal '%s'", e.Value)
			return 0, nil
		}
		return f, nil

	case *ast.FieldRefExpr:
		val := resolveNestedField(ctx.Element, e.Path)
		f, ok := exprToFloat64(val)
		if !ok {
			ctx.addWarning("field 'item.%s' resolved to non-numeric value '%v'", e.Path, val)
			return 0, nil
		}
		return f, nil

	case *ast.IntraRefExpr:
		val, exists := ctx.ComputedFields[e.Name]
		if !exists {
			ctx.addWarning("reference to undefined field '%s' (forward reference or not defined)", e.Name)
			return 0, nil
		}
		f, ok := exprToFloat64(val)
		if !ok {
			ctx.addWarning("intra-map reference '%s' resolved to non-numeric value '%v'", e.Name, val)
			return 0, nil
		}
		return f, nil

	case *ast.VarRefExpr:
		resolved := ctx.Vars.Interpolate("{{" + e.Name + "}}")
		// If it didn't resolve (still contains {{ }}), the variable is undefined
		if resolved == "{{"+e.Name+"}}" || resolved == "" {
			ctx.addWarning("variable '{{%s}}' is undefined or empty", e.Name)
			return 0, nil
		}
		f, err := strconv.ParseFloat(strings.TrimSpace(resolved), 64)
		if err != nil {
			ctx.addWarning("variable '{{%s}}' resolved to non-numeric value '%s'", e.Name, resolved)
			return 0, nil
		}
		return f, nil

	case *ast.CallExpr:
		argVal, err := EvalExpr(e.Arg, ctx)
		if err != nil {
			return 0, err
		}
		return applyMathFunc(e.Name, argVal, ctx)

	case *ast.BinaryExpr:
		left, err := EvalExpr(e.Left, ctx)
		if err != nil {
			return 0, err
		}
		right, err := EvalExpr(e.Right, ctx)
		if err != nil {
			return 0, err
		}
		return evalBinaryOp(e.Op, left, right, ctx)

	case *ast.UnaryExpr:
		val, err := EvalExpr(e.Operand, ctx)
		if err != nil {
			return 0, err
		}
		return -val, nil

	case *ast.GroupExpr:
		return EvalExpr(e.Inner, ctx)

	default:
		return 0, fmt.Errorf("unknown expression type: %T", expr)
	}
}

// applyMathFunc applies a math/coercion function to a value.
func applyMathFunc(name string, val float64, ctx *ExprContext) (float64, error) {
	switch name {
	case "number":
		return val, nil // identity for numeric values
	case "floor":
		return math.Floor(val), nil
	case "round":
		return roundHalfAwayFromZero(val), nil
	case "ceil":
		return math.Ceil(val), nil
	case "abs":
		return math.Abs(val), nil
	case "string":
		return val, nil // in arithmetic context, string() just passes through
	case "bool":
		if val != 0 {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("unknown function '%s'", name)
	}
}

// roundHalfAwayFromZero implements rounding with half away from zero.
// e.g., 2.5 → 3, -2.5 → -3
func roundHalfAwayFromZero(val float64) float64 {
	if val >= 0 {
		return math.Floor(val + 0.5)
	}
	return math.Ceil(val - 0.5)
}

// evalBinaryOp performs a binary arithmetic operation.
func evalBinaryOp(op string, left, right float64, ctx *ExprContext) (float64, error) {
	switch op {
	case "+":
		result := left + right
		checkOverflow(result, ctx)
		return result, nil
	case "-":
		result := left - right
		checkOverflow(result, ctx)
		return result, nil
	case "*":
		result := left * right
		checkOverflow(result, ctx)
		return result, nil
	case "/":
		if right == 0 {
			ctx.addWarning("division by zero")
			return 0, nil
		}
		result := left / right
		checkOverflow(result, ctx)
		return result, nil
	default:
		return 0, fmt.Errorf("unknown operator '%s'", op)
	}
}

// checkOverflow logs a warning if the result exceeds float64 range.
func checkOverflow(val float64, ctx *ExprContext) {
	if math.IsInf(val, 0) {
		ctx.addWarning("arithmetic overflow (result is infinity)")
	}
}

// exprToFloat64 converts an interface{} value to float64 for arithmetic.
func exprToFloat64(val interface{}) (float64, bool) {
	if val == nil {
		return 0, false
	}
	switch v := val.(type) {
	case float64:
		return v, true
	case int64:
		return float64(v), true
	case int:
		return float64(v), true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return 0, false
		}
		return f, true
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	default:
		return 0, false
	}
}

// addWarning adds a warning message to the context.
func (ctx *ExprContext) addWarning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	ctx.Warnings = append(ctx.Warnings, fmt.Sprintf("field '%s': %s", ctx.FieldName, msg))
}

// FormatExpr converts an expression AST back to source text (for round-trip formatting).
func FormatExpr(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	switch e := expr.(type) {
	case *ast.NumberLitExpr:
		return e.Value
	case *ast.FieldRefExpr:
		return "item." + e.Path
	case *ast.IntraRefExpr:
		return e.Name
	case *ast.VarRefExpr:
		return "{{" + e.Name + "}}"
	case *ast.CallExpr:
		return e.Name + "(" + FormatExpr(e.Arg) + ")"
	case *ast.BinaryExpr:
		return FormatExpr(e.Left) + " " + e.Op + " " + FormatExpr(e.Right)
	case *ast.UnaryExpr:
		return "-" + FormatExpr(e.Operand)
	case *ast.GroupExpr:
		return "(" + FormatExpr(e.Inner) + ")"
	default:
		return ""
	}
}
