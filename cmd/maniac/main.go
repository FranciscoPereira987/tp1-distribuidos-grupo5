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
	conn *dood.DooD
	killable []string
	gen *rand.Rand
	targets int
}

func (m *Maniac) shuffleFunc() func(i, j int) {
	return func(i, j int) {
		m.killable[i], m.killable[j] = m.killable[j], m.killable[i]
	}
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
	m.gen.Shuffle(len(m.killable), m.shuffleFunc())
	for _, toKill := range m.killable[:m.targets] {
		if err := m.conn.Kill(toKill); err != nil {
			logrus.Infof("action: killing container %s | status: failed | reason: %s", toKill, err)
		}
	}
}



func parseConfig(v *viper.Viper) (m *Maniac, err error){
	m, err = NewManiac(v.GetStringSlice("targets"), v.GetInt("kill-number"))

	return
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
	interval, err := time.ParseDuration(v.GetString("interval"))
	if err != nil {
		logrus.Fatalf("action: parsing interval | status: failed | reason: %s", err)
	}
loop:
	for {
		select {
		case <- ch:
			break loop
		default:
		}
		logrus.Info("action: Killing | status: sleeping for a bit")
		time.Sleep(interval)
		logrus.Info("action: killing | status: grabbing a gun")
		m.KillTargets()
		logrus.Info("action: killing | status: perfect crime")
	}

	logrus.Info("action: killing | status: caught by the police")
}