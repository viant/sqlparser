package query

//List represents a list
type List []*Item

func (l *List) Append(item *Item) {
	*l = append(*l, item)
}
