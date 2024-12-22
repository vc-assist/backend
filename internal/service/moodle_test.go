package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeMoodleUsername(t *testing.T) {
	table := []struct {
		input    string
		expected string
	}{
		{input: "shengzhi.hu", expected: "shengzhi.hu"},
		{input: " Shengzhi.Hu", expected: "shengzhi.hu"},
		{input: "Shengzhi.Hu@warriorlife.net\t\n", expected: "shengzhi.hu"},
		{input: "   shengzhi.hu@warriorlife.net    ", expected: "shengzhi.hu"},
	}

	for _, row := range table {
		result := NormalizeMoodleUsername(row.input)
		require.Equal(t, row.expected, result)
	}
}
