package list_test

import (
	"testing"

	"github.com/chaewonkong/matchmaker/services/apiserver/list"
	"github.com/stretchr/testify/assert"
)

type mockStruct struct {
	name string
}

func TestList(t *testing.T) {
	s := mockStruct{
		name: "Leon",
	}

	l := list.New[mockStruct]()

	assert.Equal(t, 0, l.Len())
	l.Push(s)
	assert.Equal(t, 1, l.Len())

	v, ok := l.Pop()
	assert.True(t, ok)
	assert.Equal(t, 0, l.Len())
	assert.Equal(t, s, v)

	_, ok = l.Pop()
	assert.False(t, ok)
}
