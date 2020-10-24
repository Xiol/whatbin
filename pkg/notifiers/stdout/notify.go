package stdout

import (
	"fmt"
	"strings"
)

type Notify struct{}

func New() *Notify {
	return &Notify{}
}

func (n *Notify) Notify(bins []string) error {
	fmt.Printf("Bins out today: %s\n", strings.Join(bins, ", "))
	return nil
}
