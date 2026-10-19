package levels

import "testing"

// BenchmarkLevelStep times one simulation step of each level, mid-game.
// Run with: go test ./levels/ -run '^$' -bench LevelStep
func BenchmarkLevelStep(b *testing.B) {
	for k, l := range All {
		b.Run(l.Name, func(b *testing.B) {
			s := setup(l)
			solutions[k](s)
			for f := 0; f < 120; f++ {
				l.Step(s, f)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				l.Step(s, 120+i)
			}
		})
	}
}
