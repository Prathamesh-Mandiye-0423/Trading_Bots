package decimal

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

const Precision = 8
const Scale = int64(100_000_000)

var (
	bigScale = big.NewInt(Scale)
	bigZero  = big.NewInt(0)
	bigHalf  = big.NewInt(Scale / 2)
)

type Decimal struct {
	value int64
}

func FromInt(n int64) Decimal {
	if n > math.MaxInt64/Scale || n < math.MinInt64/Scale {
		panic("decimal: out of range")
	}
	return Decimal{value: n * Scale}
}

func FromString(s string) (Decimal, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Decimal{}, fmt.Errorf("decimal: empty string")
	}
	negative := false
	if s[0] == '-' {
		negative = true
		s = s[1:]
	}

	parts := strings.SplitN(s, ".", 2)
	intPart, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return Decimal{}, fmt.Errorf("decimal: invalid integer part %q", parts[0])
	}
	if intPart > math.MaxInt64/Scale || intPart < math.MinInt64/Scale {
		return Decimal{}, fmt.Errorf("decimal: value out of range")
	}

	var fracPart int64
	if len(parts) == 2 {
		fracStr := parts[1]
		if len(fracStr) > Precision {
			return Decimal{}, fmt.Errorf("decimal: too many decimal places")
		}
		fracStr = fracStr + strings.Repeat("0", Precision-len(fracStr))
		fracPart, err = strconv.ParseInt(fracStr, 10, 64)
		if err != nil {
			return Decimal{}, fmt.Errorf("decimal: invalid fractional part")
		}
	}

	result := intPart*Scale + fracPart
	if negative {
		result = -result
	}
	return Decimal{value: result}, nil

}

func MustFromString(s string) Decimal {
	d, err := FromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

func FromRaw(raw int64) Decimal {
	return Decimal{value: raw}
}

// Arithmetic Operations

func (d Decimal) Add(other Decimal) Decimal {
	return Decimal{value: d.value + other.value}
}

func (d Decimal) Sub(other Decimal) Decimal {
	return Decimal{value: d.value - other.value}
}

func (d Decimal) Mul(other Decimal) Decimal {
	res := mulDiv(d.value, other.value, Scale)
	return Decimal{value: res}
}

func (d Decimal) Div(other Decimal) Decimal {
	if other.value == 0 {
		panic("decimal: division by zero")
	}
	res := mulDiv(d.value, Scale, other.value)
	return Decimal{value: res}
}

func mulDiv(a, b, c int64) int64 {
	ba := big.NewInt(a)
	bb := big.NewInt(b)

	var bc *big.Int
	var halfC *big.Int

	// OPTIMIZATION: If we are dividing by the standard Scale, use the pre-allocated big.Ints
	if c == Scale {
		bc = bigScale
		halfC = bigHalf
	} else {
		// Fallback for custom divisors
		bc = big.NewInt(c)
		halfC = new(big.Int).Div(new(big.Int).Abs(bc), big.NewInt(2))
	}

	// Calculate Product: intermediate can be up to ~128 bits
	prod := new(big.Int).Mul(ba, bb)

	// Rounding logic: |prod| + |c/2| / |c| (Round half away from zero)
	absProd := new(big.Int).Abs(prod)
	absProd.Add(absProd, halfC)
	quot := new(big.Int).Div(absProd, new(big.Int).Abs(bc))

	// Restore the mathematical sign
	// If the total number of negative inputs is odd, the result is negative
	if (a < 0) != (b < 0) != (c < 0) {
		quot.Neg(quot)
	}

	return quot.Int64()
}

// Comparison and Accessors
func (d Decimal) Equal(other Decimal) bool   { return d.value == other.value }
func (d Decimal) Less(other Decimal) bool    { return d.value < other.value }
func (d Decimal) Greater(other Decimal) bool { return d.value > other.value }
func (d Decimal) IsZero() bool               { return d.value == 0 }
func (d Decimal) Raw() int64                 { return d.value }

func (d Decimal) String() string {
	absVal := d.value
	sign := ""
	if absVal < 0 {
		sign = "-"
		absVal = -absVal
	}

	intPart := absVal / Scale
	fracPart := absVal % Scale
	return fmt.Sprintf("%s%d.%08d", sign, intPart, fracPart)
}

// ---- JSON ----

func (d Decimal) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.String() + `"`), nil
}

func (d *Decimal) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	parsed, err := FromString(s)
	if err != nil {
		return err
	}
	d.value = parsed.value
	return nil
}

// More convenience methods
func (d Decimal) IsPositive() bool             { return d.value > 0 }
func (d Decimal) IsNegative() bool             { return d.value < 0 }
func (d Decimal) LessEq(other Decimal) bool    { return d.value <= other.value }
func (d Decimal) GreaterEq(other Decimal) bool { return d.value >= other.value }

func Min(a, b Decimal) Decimal {
	if a.value < b.value {
		return a
	}
	return b
}

func Max(a, b Decimal) Decimal {
	if a.value > b.value {
		return a
	}
	return b
}

// Float64 converts the Decimal to a float64.
// NOTE: Only use this for display, charting, or candle aggregation.
// Never use float64 for order matching or financial arithmetic.
func (d Decimal) Float64() float64 {
	return float64(d.value) / float64(Scale)
}
