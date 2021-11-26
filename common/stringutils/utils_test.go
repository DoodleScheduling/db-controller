package stringutils

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestContainsString(t *testing.T) {
	g := NewWithT(t)
	t.Run("returns true if slice contains string", func(t *testing.T) {
		res := ContainsString([]string{"foo", "bar"}, "bar")
		g.Expect(res).To(Equal(true))
	})

	t.Run("returns false if slice does not contain", func(t *testing.T) {
		res := ContainsString([]string{"foo", "fooo"}, "bar")
		g.Expect(res).To(Equal(false))
	})
}

func TestRemoveString(t *testing.T) {
	g := NewWithT(t)
	t.Run("resliced if string was found and removed", func(t *testing.T) {
		res := RemoveString([]string{"foo", "bar"}, "bar")
		g.Expect(res).To(Equal([]string{"foo"}))
	})

	t.Run("don't reslice if string not found", func(t *testing.T) {
		res := RemoveString([]string{"foo", "bar"}, "fooo")
		g.Expect(res).To(Equal([]string{"foo", "bar"}))
	})
}
