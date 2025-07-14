package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	rocketChar = "↑"
	colors     = []lipgloss.Color{
		lipgloss.Color("226"), // Yellow
		lipgloss.Color("208"), // Orange
		lipgloss.Color("196"), // Red
		lipgloss.Color("87"),  // Light Blue
		lipgloss.Color("201"), // Magenta
		lipgloss.Color("46"),  // Green
	}
)

// Represents a single particle in a firework explosion.
type particle struct {
	x, y     float64
	vx, vy   float64
	lifespan int
	char     string
	color    lipgloss.Color
}

// Represents a firework rocket.
type rocket struct {
	x, y  int
	vy    float64
	char  string
	color lipgloss.Color
}

// The main model for our application.
type model struct {
	width     int
	height    int
	mouseX    int
	mouseY    int
	rockets   []rocket
	particles []particle
	quitting  bool
}

// A message to signal a tick in our animation.
type tickMsg time.Time

// A message to create a new firework.
type newFireworkMsg struct{}

// Creates a new firework at a random location at the bottom of the screen.
func newFirework() tea.Cmd {
	// Schedule the next firework at a random interval
	return tea.Tick(time.Duration(rand.Intn(1000)+100)*time.Millisecond, func(t time.Time) tea.Msg {
		return newFireworkMsg{}
	})
}

// Sends a tick message every frame for animation updates.
func tick() tea.Cmd {
	return tea.Tick(time.Second/15, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Returns the initial model.
func initialModel() model {
	return model{
		mouseX:    -1, // Initialize cursor off-screen
		mouseY:    -1,
		rockets:   []rocket{},
		particles: []particle{},
	}
}

// The Init function is called when the program starts.
func (m model) Init() tea.Cmd {
	// Start the animation ticker and launch the first firework
	return tea.Batch(tick(), newFirework())
}

// The Update function is called when a message is received.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}

	case tea.MouseMsg:
		// Update mouse position on any mouse event
		m.mouseX = msg.X
		m.mouseY = msg.Y

		switch msg.Button {
		case tea.MouseButtonLeft:
			if m.width == 0 || m.height == 0 {
				return m, nil
			}
			r := rocket{
				x:     msg.X,
				y:     m.height - 1,
				vy:    -1.5,
				char:  rocketChar,
				color: colors[rand.Intn(len(colors))],
			}
			m.rockets = append(m.rockets, r)
		}
		return m, nil

	case newFireworkMsg:
		// Don't create fireworks until we know the screen size.
		if m.width == 0 || m.height == 0 {
			return m, newFirework()
		}
		// Create a new rocket when a newFireworkMsg is received
		r := rocket{
			x:     rand.Intn(m.width),
			y:     m.height - 1,
			vy:    -1.5,
			char:  rocketChar,
			color: colors[rand.Intn(len(colors))],
		}
		m.rockets = append(m.rockets, r)
		// Schedule the next firework
		return m, newFirework()

	case tickMsg:
		// Don't run animation logic until we know the screen size.
		if m.width == 0 || m.height == 0 {
			return m, tick()
		}

		var updatedRockets []rocket
		for _, r := range m.rockets {
			// Check if the rocket should explode
			// It explodes if it reaches a certain height or randomly
			if r.y < m.height/3 || (r.y < m.height*2/3 && rand.Float64() < 0.1) {
				// Explode! Create particles
				numParticles := rand.Intn(20) + 30 // 30 to 49 particles
				for i := 0; i < numParticles; i++ {
					angle := (2 * math.Pi / float64(numParticles)) * float64(i)
					speed := rand.Float64()*2.5 + 1.0
					p := particle{
						x:        float64(r.x),
						y:        float64(r.y),
						vx:       math.Cos(angle) * speed,
						vy:       math.Sin(angle) * speed * 0.5,
						lifespan: rand.Intn(20) + 15,
						char:     "*",
						color:    r.color,
					}
					m.particles = append(m.particles, p)
				}
			} else {
				// Otherwise, keep the rocket moving upwards
				r.y += int(r.vy)
				updatedRockets = append(updatedRockets, r)
			}
		}
		m.rockets = updatedRockets

		var updatedParticles []particle
		for _, p := range m.particles {
			p.x += p.vx
			p.y += p.vy
			p.vy += 0.08
			p.lifespan--

			// Keep the particle if it's still alive
			if p.lifespan > 0 {
				updatedParticles = append(updatedParticles, p)
			}
		}
		m.particles = updatedParticles

		// Continue the animation ticker
		return m, tick()
	}

	return m, nil
}

// The View function is called to render the UI.
func (m model) View() string {
	if m.quitting {
		return "Bye! Thanks for watching the show.\n"
	}

	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Create a 2D slice to act as a screen buffer.
	// It stores the final, styled string for each cell.
	buffer := make([][]string, m.height)
	for i := range buffer {
		buffer[i] = make([]string, m.width)
		for j := range buffer[i] {
			buffer[i][j] = " "
		}
	}

	// Draw rockets into the buffer
	for _, r := range m.rockets {
		if r.y >= 0 && r.y < m.height && r.x >= 0 && r.x < m.width {
			style := lipgloss.NewStyle().Foreground(r.color)
			buffer[r.y][r.x] = style.Render(r.char)
		}
	}

	// Draw particles into the buffer
	for _, p := range m.particles {
		row, col := int(p.y), int(p.x)
		if row >= 0 && row < m.height && col >= 0 && col < m.width {
			// Fade out particles as they die
			alpha := float64(p.lifespan) / 35.0
			if alpha < 0.5 {
				p.char = "."
			}
			if alpha < 0.2 {
				p.char = " "
			}

			style := lipgloss.NewStyle().Foreground(p.color)
			buffer[row][col] = style.Render(p.char)
		}
	}

	// Draw the mouse cursor as a rocket, ensuring it's on top
	if m.mouseX >= 0 && m.mouseX < m.width && m.mouseY >= 0 && m.mouseY < m.height {
		// Use a distinct style for the cursor (bright white)
		cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
		// Make sure not to draw over the quit message line
		if m.mouseY < m.height-1 {
			buffer[m.mouseY][m.mouseX] = cursorStyle.Render(rocketChar)
		}
	}

	var b strings.Builder

	for i := 0; i < m.height-1; i++ {
		b.WriteString(strings.Join(buffer[i], ""))
		b.WriteString("\n")
	}
	quitMsg := "Click to launch a firework! Press 'q' to quit."
	b.WriteString(lipgloss.NewStyle().Faint(true).Render(quitMsg))

	return b.String()
}

func main() {
	rand.New(rand.NewSource(time.Now().UnixNano()))

	p := tea.NewProgram(initialModel(), tea.WithAltScreen(), tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Kaboom, there's been an error: %v", err)
		os.Exit(1)
	}
}
