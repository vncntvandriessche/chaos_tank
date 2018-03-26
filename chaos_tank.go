package main

// Example game baltantly stolen from github.com/TerrySolar/termtank

import (
	"flag"
	"fmt"
	"math/rand"
	"time"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	tl "github.com/JoelOtter/termloop"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"

	"github.com/TerrySolar/termtank/tank"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// FPS -> Frames per second
	FPS = 60
	// NS -> Namespace
	NS = "default"
)

// Tank directions
const (
	UP    int = 1
	DOWN  int = 2
	LEFT  int = 3
	RIGHT int = 4
)

var (
	masterURL  string
	kubeconfig string
	cfg        *restclient.Config
	player     Player
	enemy      Enemy
	countTime  float64
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}

// Player represents the player's tank
type Player struct {
	*tank.Tank
	preX  int
	preY  int
	level *tl.BaseLevel
}

// Enemy is an enemy tank
type Enemy struct {
	*tank.Tank
	preX       int
	preY       int
	level      *tl.BaseLevel
	status     int // 1:normal 0:dead
	pod        *core_v1.Pod
	kubeClient *kubernetes.Clientset
}

// Tick implements the Tick-event of the tl.Drawable interface
func (p *Player) Tick(event tl.Event) {
	if event.Type == tl.EventKey {
		p.preX, p.preY = p.Position()

		var bulletX, bulletY, bulletDirection int
		bulletDirection = p.Tank.GetDirection()
		cell := tl.Cell{Fg: tl.ColorRed, Bg: tl.ColorRed}

		// Handle events
		switch event.Key {
		case tl.KeyArrowUp:
			tank.TankUp(p.Tank, cell)
			p.SetPosition(p.preX, p.preY-1)
		case tl.KeyArrowDown:
			tank.TankDown(p.Tank, cell)
			p.SetPosition(p.preX, p.preY+1)
		case tl.KeyArrowRight:
			tank.TankRight(p.Tank, cell)
			p.SetPosition(p.preX+1, p.preY)
		case tl.KeyArrowLeft:
			tank.TankLeft(p.Tank, cell)
			p.SetPosition(p.preX-1, p.preY)
		case tl.KeySpace:
			switch bulletDirection {
			case tank.UP:
				bulletX = p.preX + 4
				bulletY = p.preY
			case tank.DOWN:
				bulletX = p.preX + 4
				bulletY = p.preY + 9
			case tank.LEFT:
				bulletX = p.preX
				bulletY = p.preY + 4
			case tank.RIGHT:
				bulletX = p.preX + 9
				bulletY = p.preY + 4
			}

			b := tank.NewBullet(bulletX, bulletY, bulletDirection)
			p.level.AddEntity(b)
		}
	}
}

// Collide implements the Tick-event of the tl.Physical interface
func (e *Enemy) Collide(collision tl.Physical) {
	k := *e.kubeClient
	if _, ok := collision.(tank.Bullet); ok {
		// set dead status
		e.status = 0
		fmt.Printf("Should have killed pod: \"%s\"", e.pod.GetName())
		k.CoreV1().Pods(NS).Delete(e.pod.GetName(), &meta_v1.DeleteOptions{})
		// remove from screen
		e.level.RemoveEntity(e)
	} else if _, ok := collision.(tank.Tank); ok {
		e.SetPosition(e.preX, e.preY)
	}
}

// Draw implements the Draw-event of the tl.Drawable interface for an enemy
func (e *Enemy) Draw(screen *tl.Screen) {
	countTime += screen.TimeDelta()
	e.preX, e.preY = e.Position()
	rand.Seed(time.Now().UnixNano())

	step := 3
	if countTime > 0.8 {
		direction := rand.Intn(4)
		cell := tl.Cell{Bg: tl.ColorBlue}

		switch direction + 1 {
		case UP:
			tank.TankUp(e.Tank, cell)
			e.SetPosition(e.preX, e.preY-step)
		case DOWN:
			tank.TankDown(e.Tank, cell)
			e.SetPosition(e.preX, e.preY+step)
		case LEFT:
			tank.TankLeft(e.Tank, cell)
			e.SetPosition(e.preX-step, e.preY)
		case RIGHT:
			tank.TankRight(e.Tank, cell)
			e.SetPosition(e.preX+step, e.preY)
		}

		// reset countTime
		countTime = 0.0
		tX, tY := e.Position()
		sX, sY := screen.Size()

		if tX < 0 {
			e.SetPosition(tX+step, tY)
		}
		if tX > sX-9 {
			e.SetPosition(tX-step, tY)
		}
		if tY < 0 {
			e.SetPosition(tX, tY+step)
		}
		if tY > sY-9 {
			e.SetPosition(tX, tY-step)
		}

	}
	e.Entity.Draw(screen)
}

// Draw implements the Draw-event of the tl.Drawable interface for an enemy
func (p *Player) Draw(screen *tl.Screen) {
	tX, tY := p.Position()
	sX, sY := screen.Size()

	if tX < 0 {
		p.SetPosition(tX+1, tY)
	}
	if tX > sX-9 {
		p.SetPosition(tX-1, tY)
	}
	if tY < 0 {
		p.SetPosition(tX, tY+1)
	}
	if tY > sY-9 {
		p.SetPosition(tX, tY-1)
	}
	p.Entity.Draw(screen)
}

func main() {
	// Build configuration based on passed parameters
	flag.Parse()
	fmt.Printf("Building configuration based on passed arguments...")
	cfg, _ = clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	fmt.Println(" Done")

	// Create a new client based on the configuration fetched earlier
	fmt.Printf("Creating new client from configuration...")
	kubeClient, _ := kubernetes.NewForConfig(cfg)
	fmt.Println(" Done")

	game := tl.NewGame()
	// BaseLevel
	level := tl.NewBaseLevel(tl.Cell{})

	// Initial player tank
	player := Player{
		Tank:  tank.NewTankXY(50, 80, tl.Cell{Bg: tl.ColorRed}),
		level: level,
	}
	level.AddEntity(&player)

	// Build a list of enemies based on the current nodes
	fmt.Printf("Creating enemies from pods (ns: %s)...", NS)
	enemies := make([]Enemy, 0)
	podList, _ := kubeClient.CoreV1().Pods(NS).List(meta_v1.ListOptions{})
	for i := range podList.Items {
		pod := podList.Items[i]
		e := Enemy{
			Tank:       tank.NewTankXY((120 - (i * 20)), (60 + (i * 10)), tl.Cell{Bg: tl.ColorBlue}),
			level:      level,
			status:     1,
			pod:        &pod,
			kubeClient: kubeClient,
		}
		enemies = append(enemies, e)
		level.AddEntity(&e)
		if len(enemies) > 0 {
			fmt.Println()
		}
		fmt.Printf("- %s", pod.GetName())
	}
	if len(enemies) > 0 {
		fmt.Println("\n...Done")
	} else {
		fmt.Println(" Done")
	}

	// Start the game
	game.Screen().SetLevel(level)
	game.Screen().EnablePixelMode()
	game.Screen().SetFps(FPS)
	game.Start()
}
