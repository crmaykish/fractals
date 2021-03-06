package fractal_core

import (
	"math"
	"math/cmplx"
	"sync"
)

const DefaultZoomLevel = 0.5
const DefaultMaxIterations = 1000
const mandelbrotEscapeRadius = 2.0

type Mandelbrot struct {
	ImageWidth             int
	ImageHeight            int
	center                 complex128
	zoomLevel              float64
	maxIterations          int
	buffer                 [][]uint32
	minX, minY, maxX, maxY float64
	histogram              []uint32
	hue                    [][]float64
}

func Create(width, height int, center complex128) *Mandelbrot {
	// Create the main struct
	m := Mandelbrot{ImageWidth: width, ImageHeight: height, center: center}

	// Set up default configuration
	SetMaxIterations(&m, DefaultMaxIterations)
	SetZoom(&m, DefaultZoomLevel)

	// Create a buffer to store all pixels
	m.buffer = make([][]uint32, width)
	for i := 0; i < width; i++ {
		m.buffer[i] = make([]uint32, height)
	}

	return &m
}

func Generate(m *Mandelbrot) {
	m.histogram = make([]uint32, m.maxIterations)

	m.hue = make([][]float64, m.ImageWidth)
	for i := 0; i < m.ImageWidth; i++ {
		m.hue[i] = make([]float64, m.ImageHeight)
	}

	var wg sync.WaitGroup

	for x := 0; x < m.ImageWidth; x++ {
		for y := 0; y < m.ImageHeight; y++ {
			// Map this pixel to a complex number on the plane
			var a = MapIntToFloat(x, 0, m.ImageWidth, m.minX, m.maxX)
			var b = MapIntToFloat(y, 0, m.ImageHeight, m.minY, m.maxY)

			// p is a complex number of the form a+bi
			var p = complex(a, b)

			// Check if this point is in the Mandelbrot set
			wg.Add(1)
			go func(x, y int) {
				iterations := pointInSet(p, m.maxIterations)

				// The number of iterations this point endured is returned and stored in the blob array
				m.buffer[x][y] = uint32(iterations)

				// Increment the histogram with the iteration result
				if iterations != m.maxIterations {
					m.histogram[iterations]++
				}

				wg.Done()
			}(x, y)
		}
	}

	wg.Wait()

	var total uint32 = 0

	// Generate the histogram
	for i := 0; i < m.maxIterations; i++ {
		total += m.histogram[i]
	}

	// Find a hue for each point in the array
	for x := 0; x < m.ImageWidth; x++ {
		for y := 0; y < m.ImageHeight; y++ {

			var v = m.buffer[x][y]
			for i := 0; i < int(v); i++ {
				m.hue[x][y] += float64(m.histogram[i]) / float64(total)
			}
		}
	}

}

func SetCenter(m *Mandelbrot, center complex128) {
	m.center = center
}

func SetZoom(m *Mandelbrot, z float64) {
	m.zoomLevel = z

	offset := 1.0 / m.zoomLevel
	stretch := float64(m.ImageHeight) / float64(m.ImageWidth)

	// Set the range of the X axis
	m.minX = real(m.center) - offset
	m.maxX = real(m.center) + offset

	// Set the range of the Y access
	// Account for vertical stretch due to non-square image size
	m.minY = imag(m.center) - offset*stretch
	m.maxY = imag(m.center) + offset*stretch
}

func ScaleZoom(m *Mandelbrot, scale float64) {
	SetZoom(m, m.zoomLevel*scale)
}

// Return x min, y min, x max, x max of the current view
func GetBounds(m *Mandelbrot) (float64, float64, float64, float64) {
	return m.minX, m.minY, m.maxX, m.maxY
}

func GetBuffer(m *Mandelbrot) [][]uint32 {
	return m.buffer
}

func GetZoom(m *Mandelbrot) float64 {
	return m.zoomLevel
}

func GetMaxIterations(m *Mandelbrot) int {
	return m.maxIterations
}

func SetMaxIterations(m *Mandelbrot, i int) {
	m.maxIterations = i

	// remake the histogram
	m.histogram = make([]uint32, m.maxIterations)
}

func GetHistogram(m *Mandelbrot) []uint32 {
	return m.histogram
}

func GetHue(m *Mandelbrot) [][]float64 {
	return m.hue
}

// Check if the given complex number is in the Mandelbrot set
// If it is, return maxIterations; if not, return the number of iterations
// it took to diverge outside of the escape radius
func pointInSet(val complex128, maxIterations int) int {
	// Split the complex number into real and imaginary parts
	x := real(val)
	y := imag(val)

	// If the given point is in the main cardioid or the period 2 bulb,
	// it's definitely in the set. No need to iterate on it.
	// This is a huge optimization for points near the main cardioid
	if pointInCardioid(x, y) || pointInPeriod2Bulb(x, y) {
		return maxIterations
	}

	// Keep track of the last two iterated points. If the current
	// point has already been seen, it cannot diverge and must be
	// in the set.
	// TODO: Look into generalizing this instead of just keeping
	// track of 2 points. See where the best tradeoff is
	last0 := complex(0, 0)
	last1 := complex(0, 0)

	// Current value of the point under iteration
	var curr complex128

	// Iterate the given point through fc(z) = z^2 + c until it
	// diverges outside of the set or the max iteration has been reached
	for i := 0; i < maxIterations; i++ {
		// Put the current point through the equation
		curr = cmplx.Pow(curr, 2) + val

		if curr == last0 || curr == last1 {
			// If we've seen this point before, it must be in the set
			return maxIterations
		}

		if cmplx.Abs(curr) > mandelbrotEscapeRadius {
			// Point diverged, return the number of iterations it took
			return i
		}

		// Update the last points before iterating again
		last1 = last0
		last0 = curr
	}

	// Point did not diverge, assume it's in the set
	return maxIterations
}

func pointInCardioid(a, b float64) bool {
	p := math.Sqrt(math.Pow(a-(0.25), 2) + math.Pow(b, 2))
	comp := p - 2*math.Pow(p, 2) + (0.25)
	return a <= comp
}

func pointInPeriod2Bulb(a, b float64) bool {
	return math.Pow(a+1, 2)+math.Pow(b, 2) <= float64(1)/float64(16)
}
