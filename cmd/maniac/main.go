package main

import (
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/dood"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Maniac struct {
	conn     *dood.DooD
	killable []string
	gen      *rand.Rand
	targets  int
}

func (m *Maniac) shuffleFunc(i, j int) {
	m.killable[i], m.killable[j] = m.killable[j], m.killable[i]
}

func NewManiac(possibleTargets []string, targets int) (m *Maniac, err error) {
	logrus.Infof("targets: %s", possibleTargets)
	var conn *dood.DooD
	conn, err = dood.NewDockerClientDefault()
	if err != nil {
		return
	}
	m = new(Maniac)
	m.conn = conn
	seed := rand.NewSource(time.Now().Unix())
	m.killable = possibleTargets
	m.gen = rand.New(seed)
	m.targets = targets
	return
}

func (m *Maniac) KillTargets() {
	m.gen.Shuffle(len(m.killable), m.shuffleFunc)
	for _, toKill := range m.killable[:m.targets] {
		if err := m.conn.Kill(toKill); err != nil {
			logrus.Infof("action: killing container %s | status: failed | reason: %s", toKill, err)
		}
	}
}

func parseConfig(v *viper.Viper) (*Maniac, error) {
	return NewManiac(v.GetStringSlice("targets"), v.GetInt("kill"))
}

func main() {
	v, err := utils.InitConfig("crazed", "cmd/maniac")
	if err != nil {
		logrus.Fatalf("action: parsing config | status: failed | reason: %s", err)
	}

	m, err := parseConfig(v)
	if err != nil {
		logrus.Fatalf("action: initializing maniac | status: failed | reason: %s", err)
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGKILL)
	interval := v.GetDuration("interval")
	if interval <= 0 {
		logrus.Fatal("action: parsing interval | status: failed | reason: non-positive interval")
	}
loop:
	for ticker := time.Tick(interval); ; {
		logrus.Info("action: Killing | status: sleeping for a bit")
		select {
		case <-ch:
			break loop
		case <-ticker:
		}
		logrus.Info("action: killing | status: grabbing a gun")
		m.KillTargets()
		logrus.Info("action: killing | status: perfect crime")
	}

	logrus.Info("action: killing | status: caught by the police")
}
