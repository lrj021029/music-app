package viz

import (
	"cmp"
	"image"
	"slices"

	"github.com/egoist/mygo/ui"
)

// Palette returns up to k colors of img, the most of it first: the means
// of the clusters of its pixels, sampled on a grid of 32 by 32, seeded
// with its most common colors, each far from those before, and refined.
// The same image always gives the same colors; transparent pixels count
// for none.
func Palette(img image.Image, k int) []ui.Color {
	if img == nil || k < 1 {
		return nil
	}
	b := img.Bounds()
	if b.Empty() {
		return nil
	}
	const side = 32
	pts := make([][3]float32, 0, side*side)
	for y := range side {
		for x := range side {
			px := b.Min.X + (2*x+1)*b.Dx()/(2*side)
			py := b.Min.Y + (2*y+1)*b.Dy()/(2*side)
			r, g, bl, a := img.At(px, py).RGBA()
			if a < 0x8000 {
				continue
			}
			pts = append(pts, [3]float32{float32(r) / 0xffff, float32(g) / 0xffff, float32(bl) / 0xffff})
		}
	}
	if len(pts) == 0 {
		return nil
	}
	// Seeds: the most common of 8×8×8 boxes of colors, by how many pixels
	// they have and how far they are from the seeds before.
	var count [512]int
	var sum [512][3]float32
	for _, p := range pts {
		i := int(p[0]*7.99)<<6 | int(p[1]*7.99)<<3 | int(p[2]*7.99)
		count[i]++
		sum[i][0], sum[i][1], sum[i][2] = sum[i][0]+p[0], sum[i][1]+p[1], sum[i][2]+p[2]
	}
	var centers [][3]float32
	for len(centers) < k {
		best, score := -1, float32(0)
		for i, n := range count {
			if n == 0 {
				continue
			}
			m := scale(sum[i], n)
			d := float32(1)
			for _, c := range centers {
				d = min(d, dist2(m, c))
			}
			if s := float32(n) * d; s > score {
				best, score = i, s
			}
		}
		if best < 0 || len(centers) > 0 && score < float32(len(pts))*0.0005 {
			break
		}
		centers = append(centers, scale(sum[best], count[best]))
	}
	weight := make([]int, len(centers))
	acc := make([][3]float32, len(centers))
	for range 6 {
		clear(weight)
		clear(acc)
		for _, p := range pts {
			j := nearest(centers, p)
			weight[j]++
			acc[j][0], acc[j][1], acc[j][2] = acc[j][0]+p[0], acc[j][1]+p[1], acc[j][2]+p[2]
		}
		for j := range centers {
			if weight[j] > 0 {
				centers[j] = scale(acc[j], weight[j])
			}
		}
	}
	order := make([]int, 0, len(centers))
	for j := range centers {
		if weight[j] > 0 {
			order = append(order, j)
		}
	}
	slices.SortStableFunc(order, func(a, b int) int { return cmp.Compare(weight[b], weight[a]) })
	out := make([]ui.Color, len(order))
	for i, j := range order {
		c := centers[j]
		out[i] = ui.RGB(byte8(c[0]), byte8(c[1]), byte8(c[2]))
	}
	return out
}

func scale(s [3]float32, n int) [3]float32 {
	f := 1 / float32(n)
	return [3]float32{s[0] * f, s[1] * f, s[2] * f}
}

func dist2(a, b [3]float32) float32 {
	r, g, bl := a[0]-b[0], a[1]-b[1], a[2]-b[2]
	return r*r + g*g + bl*bl
}

func nearest(centers [][3]float32, p [3]float32) int {
	best, d := 0, float32(4)
	for j, c := range centers {
		if e := dist2(p, c); e < d {
			best, d = j, e
		}
	}
	return best
}

func byte8(f float32) uint8 { return uint8(min(max(f, 0), 1)*255 + 0.5) }
