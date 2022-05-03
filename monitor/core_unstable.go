//go:build core_unstable
// +build core_unstable

package monitor

func init() {
	RegisterModule(&Module{
		Name: "core_unstable",
	})
	GlobalDeepCopy = false
}
