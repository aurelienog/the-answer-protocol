package animations

import "math"

type HealthBarAnimation struct {
	Current     float64
	Target      float64
	Maximum     int
	Initialized bool
}

// SetTarget actualiza el objetivo sin interrumpir la transición actual.
func (h *HealthBarAnimation) SetTarget(current, maximum int) bool {
	if maximum <= 0 {
		return false
	}

	if current < 0 {
		current = 0
	}
	if current > maximum {
		current = maximum
	}

	// Primera actualización: mostrar directamente el valor inicial.
	if !h.Initialized || h.Maximum != maximum {
		h.Current = float64(current)
		h.Target = float64(current)
		h.Maximum = maximum
		h.Initialized = true
		return false
	}

	h.Target = float64(current)
	return h.Animating()
}

// Tick avanza la animación.
// Devuelve true mientras siga animándose.
func (h *HealthBarAnimation) Tick() bool {
	if !h.Initialized {
		return false
	}

	difference := h.Target - h.Current

	if math.Abs(difference) < 0.5 {
		h.Current = h.Target
		return false
	}

	// Suavizado exponencial.
	h.Current += difference * 0.22

	return true
}

func (h *HealthBarAnimation) Animating() bool {
	return math.Abs(h.Target-h.Current) >= 0.5
}