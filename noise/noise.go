// Package noise is numbers that look unplanned and are not.
//
// Everything that has to look natural here is built from two things. A hash
// turns integers into a number between nought and one that is the same every
// time: which way a spark flies, where a star is, which plate a hex is on.
//
// Gradient noise turns a position into a smooth hill-and-valley value: the
// grain of a disk, a coastline, a cloud.
//
// Neither has any state, and neither reads a clock or a random source. A frame
// drawn twice is the same frame, a particle can be asked where it is at any
// moment without having been asked about the moments before, and a picture
// can be tested.
//
// The hash is splitmix64's finaliser, which is good enough that neighbouring
// integers give unrelated numbers. grade.Hash2's sine trick is not, and says
// so; it stays where it is because the pictures made with it would move.
package noise

import "math"

// mix is splitmix64's finaliser: every bit of the input reaches every bit of
// the output.
func mix(z uint64) uint64 {
	z ^= z >> 30
	z *= 0xbf58476d1ce4e5b9
	z ^= z >> 27
	z *= 0x94d049bb133111eb
	return z ^ z>>31
}

// unit is the top 53 bits of a hash as a number from nought up to one.
func unit(z uint64) float64 { return float64(z>>11) / (1 << 53) }

// Hash is a number from nought up to one for a seed and an index. The same
// pair gives the same number for ever.
func Hash(seed uint64, i int64) float64 {
	return unit(mix(seed ^ mix(uint64(i)+0x9e3779b97f4a7c15)))
}

// Hash2 is Hash for a pair of indices.
func Hash2(seed uint64, i, j int64) float64 {
	j2 := mix(uint64(j) + 0x9e3779b97f4a7c15)
	return unit(mix(seed ^ mix(uint64(i)+j2)))
}

// Hash3 is Hash for three indices.
func Hash3(seed uint64, i, j, k int64) float64 {
	return unit(bits3(seed, i, j, k))
}

// bits3 is the whole hash of three indices, for a caller that wants several
// numbers out of one: a lattice corner's gradient takes its low bits.
func bits3(seed uint64, i, j, k int64) uint64 {
	return mix(seed ^ mix(uint64(i)+mix(uint64(j)+
		mix(uint64(k)+0x9e3779b97f4a7c15))))
}

// Gradient is three-dimensional gradient noise at a point: nought on every
// integer lattice point, smooth between them, and within about 0.9 of nought
// in either direction.
//
// Perlin's improved noise, with the gradient at each corner taken from the hash
// rather than a shuffled table. A table is 256 entries of one fixed shuffle;
// the hash gives every seed its own field, which is how two planets differ.
func Gradient(seed uint64, x, y, z float64) float64 {
	fx, fy, fz := math.Floor(x), math.Floor(y), math.Floor(z)
	ix, iy, iz := int64(fx), int64(fy), int64(fz)
	x, y, z = x-fx, y-fy, z-fz
	u, v, w := fade(x), fade(y), fade(z)

	g := func(dx, dy, dz int64) float64 {
		h := bits3(seed, ix+dx, iy+dy, iz+dz)
		return grad(h, x-float64(dx), y-float64(dy), z-float64(dz))
	}
	return lerp(w,
		lerp(v, lerp(u, g(0, 0, 0), g(1, 0, 0)),
			lerp(u, g(0, 1, 0), g(1, 1, 0))),
		lerp(v, lerp(u, g(0, 0, 1), g(1, 0, 1)),
			lerp(u, g(0, 1, 1), g(1, 1, 1))))
}

// fade is Perlin's quintic, whose first two derivatives are nought at both
// ends, so the lattice does not show as creases.
func fade(t float64) float64 { return t * t * t * (t*(t*6-15) + 10) }

// lerp is a straight line from a to b.
func lerp(t, a, b float64) float64 { return a + t*(b-a) }

// grad is the dot product of an offset with one of the twelve edge directions
// of a cube, chosen by the low bits of a hash.
func grad(h uint64, x, y, z float64) float64 {
	switch h % 12 {
	case 0:
		return x + y
	case 1:
		return -x + y
	case 2:
		return x - y
	case 3:
		return -x - y
	case 4:
		return x + z
	case 5:
		return -x + z
	case 6:
		return x - z
	case 7:
		return -x - z
	case 8:
		return y + z
	case 9:
		return -y + z
	case 10:
		return y - z
	}
	return -y - z
}

// Fractal is gradient noise summed over octaves, each twice the frequency and
// half the amplitude of the one before, scaled back to about the range of one
// octave. It is what makes a coastline ragged at every scale rather than
// round at one.
func Fractal(seed uint64, x, y, z float64, octaves int) float64 {
	sum, amp, norm := 0.0, 1.0, 0.0
	for o := range octaves {
		s := seed + uint64(o)*0x632be59bd9b4e019
		sum += amp * Gradient(s, x, y, z)
		norm += amp
		amp *= 0.5
		// 2.03 and not 2, so the octaves do not line up
		x, y, z = x*2.03, y*2.03, z*2.03
	}
	if norm == 0 {
		return 0
	}
	return sum / norm
}
