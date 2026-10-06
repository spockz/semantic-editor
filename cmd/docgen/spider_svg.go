// spider_svg.go renders task complexity as a self-contained seven-axis radar chart.
package main

import (
	"fmt"
	"math"
	"strings"
)

type ComplexityProfile struct {
	Locality    int `json:"locality"`
	Context     int `json:"context"`
	Ambiguity   int `json:"ambiguity"`
	Composition int `json:"composition"`
	Invariants  int `json:"invariants"`
	Scale       int `json:"scale"`
	Staging     int `json:"staging"`
}

func (profile ComplexityProfile) Validate() error {
	for _, value := range []int{profile.Locality, profile.Context, profile.Ambiguity, profile.Composition, profile.Invariants, profile.Scale, profile.Staging} {
		if value < 1 || value > 5 {
			return fmt.Errorf("complexity axes must have values from 1 to 5")
		}
	}
	return nil
}

func RenderRadarSVG(profile ComplexityProfile, width, height int) (string, error) {
	if width < 280 || height < 280 {
		return "", fmt.Errorf("radar chart dimensions must be at least 280 by 280 pixels")
	}
	if err := profile.Validate(); err != nil {
		return "", err
	}
	axes := []struct {
		name  string
		value int
	}{
		{name: "Locality", value: profile.Locality},
		{name: "Context", value: profile.Context},
		{name: "Ambiguity", value: profile.Ambiguity},
		{name: "Composition", value: profile.Composition},
		{name: "Invariants", value: profile.Invariants},
		{name: "Scale", value: profile.Scale},
		{name: "Staging", value: profile.Staging},
	}
	cx, cy := float64(width)/2, float64(height)/2
	radius := math.Min(cx, cy) - 52
	angleStep := 2 * math.Pi / float64(len(axes))
	var svg strings.Builder
	fmt.Fprintf(&svg, `<svg xmlns="http://www.w3.org/2000/svg" role="img" aria-label="Seven-axis task complexity radar chart" width="%d" height="%d" viewBox="0 0 %d %d">`, width, height, width, height)
	svg.WriteString(`<rect width="100%" height="100%" rx="8" fill="#0f172a"/>`)
	for level := 1; level <= 5; level++ {
		points := radarPoints(cx, cy, radius*float64(level)/5, angleStep, len(axes))
		fmt.Fprintf(&svg, `<polygon points="%s" fill="none" stroke="#334155" stroke-width="1"/>`, points)
	}
	var valuePoints []string
	for index, axis := range axes {
		angle := float64(index)*angleStep - math.Pi/2
		x := cx + radius*math.Cos(angle)
		y := cy + radius*math.Sin(angle)
		fmt.Fprintf(&svg, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#475569" stroke-width="1"/>`, cx, cy, x, y)
		labelRadius := radius + 34
		labelX := cx + labelRadius*math.Cos(angle)
		labelY := cy + labelRadius*math.Sin(angle)
		anchor := "middle"
		if math.Cos(angle) > 0.25 {
			anchor = "start"
		} else if math.Cos(angle) < -0.25 {
			anchor = "end"
		}
		fmt.Fprintf(&svg, `<text x="%.1f" y="%.1f" text-anchor="%s" fill="#cbd5e1" font-size="12" font-family="sans-serif">%s</text>`, labelX, labelY+4, anchor, axis.name)
		valueRadius := radius * float64(axis.value) / 5
		valuePoints = append(valuePoints, fmt.Sprintf("%.1f,%.1f", cx+valueRadius*math.Cos(angle), cy+valueRadius*math.Sin(angle)))
	}
	fmt.Fprintf(&svg, `<polygon points="%s" fill="#38bdf8" fill-opacity="0.28" stroke="#38bdf8" stroke-width="2"/>`, strings.Join(valuePoints, " "))
	svg.WriteString(`</svg>`)
	return svg.String(), nil
}

func radarPoints(cx, cy, radius, angleStep float64, count int) string {
	points := make([]string, count)
	for index := range count {
		angle := float64(index)*angleStep - math.Pi/2
		points[index] = fmt.Sprintf("%.1f,%.1f", cx+radius*math.Cos(angle), cy+radius*math.Sin(angle))
	}
	return strings.Join(points, " ")
}
