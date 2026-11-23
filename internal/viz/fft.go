package viz

import "math"

// fft is a radix-2 transform of n points, its tables made once, which
// transforms re and im in place and allocates nothing.
type fft struct {
	n        int
	rev      []int32
	cos, sin []float32 // e^(-2πik/n), for k below n/2
	re, im   []float32
}

func newFFT(n int) *fft {
	f := &fft{n: n, rev: make([]int32, n), cos: make([]float32, n/2), sin: make([]float32, n/2), re: make([]float32, n), im: make([]float32, n)}
	bits := 0
	for 1<<bits < n {
		bits++
	}
	for i := range n {
		r := 0
		for b := range bits {
			if i&(1<<b) != 0 {
				r |= 1 << (bits - 1 - b)
			}
		}
		f.rev[i] = int32(r)
	}
	for k := range n / 2 {
		a := -2 * math.Pi * float64(k) / float64(n)
		f.cos[k], f.sin[k] = float32(math.Cos(a)), float32(math.Sin(a))
	}
	return f
}

func (f *fft) transform() {
	re, im := f.re, f.im
	for i, j := range f.rev {
		if i < int(j) {
			re[i], re[j] = re[j], re[i]
			im[i], im[j] = im[j], im[i]
		}
	}
	for size := 2; size <= f.n; size <<= 1 {
		half, step := size/2, f.n/size
		for start := 0; start < f.n; start += size {
			for k := range half {
				wr, wi := f.cos[k*step], f.sin[k*step]
				i, j := start+k, start+k+half
				tr := re[j]*wr - im[j]*wi
				ti := re[j]*wi + im[j]*wr
				re[j], im[j] = re[i]-tr, im[i]-ti
				re[i] += tr
				im[i] += ti
			}
		}
	}
}
