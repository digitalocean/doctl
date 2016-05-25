package config

type node struct {
	children []node
	key      string
	value    interface{}
}

func (n *node) add(k string, v interface{}) {

}
